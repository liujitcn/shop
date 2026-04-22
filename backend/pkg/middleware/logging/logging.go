package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"shop/api/gen/go/base"
	"shop/pkg/gen/data"
	pkgQueue "shop/pkg/queue"
	"strings"
	"time"

	_const "shop/pkg/const"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/transport/http/status"
	"github.com/liujitcn/go-utils/trans"
	"github.com/liujitcn/gorm-kit/repo"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/mileusna/useragent"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Redacter 定义日志脱敏接口。
type Redacter interface {
	Redact() string
}

// Server 创建服务端访问日志中间件。
func Server(_ log.Logger,
	baseUserRepo *data.BaseUserRepo,
	authenticator authnEngine.Authenticator,
) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			startTime := time.Now()
			// 日志信息
			baseLog := models.BaseLog{
				RequestTime: startTime,
				// 默认返回码按成功初始化，后续再根据实际错误覆盖。
				StatusCode: int32(status.FromGRPCCode(codes.OK)),
			}
			// 当前上下文存在服务端传输信息时，补充访问日志的请求元数据。
			if info, ok := transport.FromServerContext(ctx); ok {
				baseLog.Operation = info.Operation()
				var fullErr error
				// 当前请求走 HTTP 传输层时，补充 HTTP 相关访问日志字段。
				if htr, htrOk := info.(*http.Transport); htrOk {
					baseLog.RequestID = getRequestId(ctx, htr.Request())
					// 文件上传不存请求内容
					// 文件上传和下载请求体通常较大，不记录请求体内容。
					if !(htr.Operation() == base.OperationFileServiceMultiUploadFile || htr.Operation() == base.OperationFileServiceUploadFile || htr.Operation() == base.OperationFileServiceDownloadFile) {
						baseLog.RequestBody = extractArgs(req)
					}

					headers := htr.RequestHeader()
					headersMap := make(map[string]string)
					for _, key := range headers.Keys() {
						headersMap[key] = htr.RequestHeader().Get(key)
					}
					var headersBytes []byte
					// 请求头序列化成功时，再写入日志字段。
					headersBytes, fullErr = json.Marshal(headersMap)
					if fullErr == nil {
						baseLog.RequestHeader = string(headersBytes)
					}

					clientIp := getClientRealIP(htr.Request())
					referer, _ := url.QueryUnescape(htr.RequestHeader().Get(HeaderKeyReferer))
					requestUri, _ := url.QueryUnescape(htr.Request().RequestURI)

					baseLog.Method = htr.Request().Method
					baseLog.Path = htr.PathTemplate()
					baseLog.Referer = trans.Ptr(referer)
					baseLog.RequestURI = trans.Ptr(requestUri)
					baseLog.ClientIP = trans.Ptr(clientIp)
					baseLog.Location = trans.Ptr(clientIpToLocation(clientIp))

					// 登录接口优先从请求体回填用户名与用户编号。
					if htr.Operation() == base.OperationLoginServiceLogin {
						var loginRequest base.LoginRequest
						// 登录请求体可正常解析时，继续提取登录用户名。
						fullErr = json.Unmarshal([]byte(baseLog.RequestBody), &loginRequest)
						if fullErr == nil {
							userName := loginRequest.GetUserName()
							// 登录用户名存在时，继续回查基础用户信息。
							if len(userName) > 0 {
								baseLog.UserName = userName
								var baseUser *models.BaseUser
								query := baseUserRepo.Query(htr.Request().Context()).BaseUser
								opts := make([]repo.QueryOption, 0, 1)
								opts = append(opts, repo.Where(query.UserName.Eq(userName)))
								// 基础用户回查成功时，补充用户编号。
								baseUser, fullErr = baseUserRepo.Find(htr.Request().Context(), opts...)
								if fullErr == nil {
									baseLog.UserID = baseUser.ID
								}
							}
						}
					} else {
						authToken := htr.RequestHeader().Get(HeaderKeyAuthorization)
						ut := extractAuthToken(authToken, authenticator)
						// 非登录接口能解析出用户令牌时，回填登录态用户信息。
						if ut != nil {
							baseLog.UserID = ut.UserId
							baseLog.UserName = ut.UserName
						}
					}

					// 用户代理信息
					strUserAgent := htr.RequestHeader().Get(HeaderKeyUserAgent)
					ua := useragent.Parse(strUserAgent)

					var deviceName string
					// User-Agent 能识别设备名时，优先使用识别结果。
					if ua.Device != "" {
						deviceName = ua.Device
					} else {
						// 未识别设备名但属于桌面端时，统一标记为 PC。
						if ua.Desktop {
							deviceName = "PC"
						}
					}

					baseLog.UserAgent = ua.String
					baseLog.BrowserVersion = ua.Version
					baseLog.BrowserName = ua.Name
					baseLog.OsName = ua.OS
					baseLog.OsVersion = ua.OSVersion
					baseLog.ClientID = deviceName
					baseLog.ClientName = deviceName
				}
			}
			reply, err = handler(ctx, req)
			baseLog.Success = err == nil
			// 当前错误可转换为 Kratos 标准错误时，补充业务错误码和原因。
			if se := errors.FromError(err); se != nil {
				baseLog.StatusCode = se.Code
				baseLog.Reason = se.Reason
			}
			baseLog.CostTime = time.Since(startTime).Milliseconds()
			var responseBytes []byte
			var responseErr error
			// 响应体序列化成功时，写入响应日志。
			responseBytes, responseErr = json.Marshal(reply)
			if responseErr == nil {
				baseLog.Response = string(responseBytes)
			}
			level, stack := extractError(err)
			// 存在堆栈信息时，追加到日志原因字段便于排查。
			if len(stack) > 0 {
				baseLog.Reason = fmt.Sprintf("[%s]%s", baseLog.Reason, stack)
			}
			// 写入日志
			pkgQueue.AddQueue(_const.Log, &baseLog)
			logLine := buildAccessLogLine(&baseLog)
			// 错误请求使用错误级别输出，便于在控制台快速筛选异常请求。
			if level == log.LevelError {
				log.Error(logLine)
			} else {
				// 非错误请求统一输出单行文本日志，避免结构化日志过于冗长。
				log.Info(logLine)
			}
			return
		}
	}
}

// buildAccessLogLine 构造控制台单行访问日志。
func buildAccessLogLine(baseLog *models.BaseLog) string {
	return fmt.Sprintf(
		"operation=%s method=%s path=%s args=%s code=%d latency=%s",
		normalizeLogField(baseLog.Operation),
		normalizeLogField(baseLog.Method),
		normalizeLogField(baseLog.Path),
		normalizeLogField(baseLog.RequestBody),
		baseLog.StatusCode,
		fmt.Sprintf("%dms", baseLog.CostTime),
	)
}

// normalizeLogField 将日志字段压缩成单行文本。
func normalizeLogField(value string) string {
	value = strings.TrimSpace(value)
	// 空值字段统一输出占位符，避免控制台日志字段缺失。
	if value == "" {
		return "-"
	}
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.Join(strings.Fields(value), " ")
}

// extractArgs 提取请求体日志内容。
func extractArgs(req interface{}) string {
	requestBody, err := marshalRequestBody(req)
	// 请求对象能正常序列化时，统一按 JSON 写入日志，便于后台直接格式化展示。
	if err == nil {
		return string(requestBody)
	}

	// 请求对象实现脱敏接口但无法直接序列化时，回退记录脱敏后的 JSON 字符串。
	if redacter, ok := req.(Redacter); ok {
		return marshalFallbackText(redacter.Redact())
	}
	// 请求对象实现 Stringer 时，回退复用其字符串表示。
	if stringer, ok := req.(fmt.Stringer); ok {
		return marshalFallbackText(stringer.String())
	}
	return marshalFallbackText(fmt.Sprintf("%+v", req))
}

// extractError 提取错误日志级别和堆栈信息。
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}

// marshalRequestBody 将请求对象统一序列化成 JSON。
func marshalRequestBody(req interface{}) ([]byte, error) {
	// 空请求统一写成空对象，避免日志字段出现空串。
	if req == nil {
		return []byte("{}"), nil
	}

	// Proto 请求优先使用 protojson，确保 GET 参数也能落成标准 JSON。
	if message, ok := req.(proto.Message); ok {
		return protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: false,
		}.Marshal(message)
	}

	return json.Marshal(req)
}

// marshalFallbackText 将兜底文本包装成合法 JSON 字符串。
func marshalFallbackText(text string) string {
	textBytes, err := json.Marshal(strings.TrimSpace(text))
	if err != nil {
		return strings.TrimSpace(text)
	}
	return string(textBytes)
}
