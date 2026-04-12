package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"shop/api/gen/go/base"
	"shop/pkg/gen/data"
	"shop/pkg/utils"
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
)

// Redacter defines how to log an object
type Redacter interface {
	Redact() string
}

// Server is an server logging middleware.
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
				// default code
				StatusCode: int32(status.FromGRPCCode(codes.OK)),
			}
			if info, ok := transport.FromServerContext(ctx); ok {
				baseLog.Operation = info.Operation()
				var fullErr error
				if htr, htrOk := info.(*http.Transport); htrOk {
					baseLog.RequestID = getRequestId(htr.Request())
					// 文件上传不存请求内容
					if !(htr.Operation() == base.OperationFileServiceMultiUploadFile || htr.Operation() == base.OperationFileServiceUploadFile || htr.Operation() == base.OperationFileServiceDownloadFile) {
						baseLog.RequestBody = extractArgs(req)
					}

					headers := htr.RequestHeader()
					headersMap := make(map[string]string)
					for _, key := range headers.Keys() {
						headersMap[key] = htr.RequestHeader().Get(key)
					}
					var headersBytes []byte
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

					if htr.Operation() == base.OperationLoginServiceLogin {
						var loginRequest base.LoginRequest
						if fullErr = json.Unmarshal([]byte(baseLog.RequestBody), &loginRequest); fullErr == nil {
							userName := loginRequest.GetUserName()
							if len(userName) > 0 {
								baseLog.UserName = userName
								var baseUser *models.BaseUser
								baseUser, fullErr = baseUserRepo.Find(htr.Request().Context(),
									repo.Where(baseUserRepo.Query(ctx).BaseUser.UserName.Eq(userName)),
								)
								if fullErr == nil {
									baseLog.UserID = baseUser.ID
								}
							}
						}
					} else {
						authToken := htr.RequestHeader().Get(HeaderKeyAuthorization)
						ut := extractAuthToken(authToken, authenticator)
						if ut != nil {
							baseLog.UserID = ut.UserId
							baseLog.UserName = ut.UserName
						}
					}

					// 用户代理信息
					strUserAgent := htr.RequestHeader().Get(HeaderKeyUserAgent)
					ua := useragent.Parse(strUserAgent)

					var deviceName string
					if ua.Device != "" {
						deviceName = ua.Device
					} else {
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
			if se := errors.FromError(err); se != nil {
				baseLog.StatusCode = se.Code
				baseLog.Reason = se.Reason
			}
			baseLog.CostTime = time.Since(startTime).Milliseconds()
			responseBytes, responseErr := json.Marshal(reply)
			if responseErr == nil {
				baseLog.Response = string(responseBytes)
			}
			level, stack := extractError(err)
			if len(stack) > 0 {
				baseLog.Reason = fmt.Sprintf("[%s]%s", baseLog.Reason, stack)
			}
			// 写入日志
			utils.AddQueue(_const.Log, &baseLog)
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
	if value == "" {
		return "-"
	}
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.Join(strings.Fields(value), " ")
}

// extractArgs returns the string of the req
func extractArgs(req interface{}) string {
	if redacter, ok := req.(Redacter); ok {
		return redacter.Redact()
	}
	if stringer, ok := req.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", req)
}

// extractError returns the string of the error
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}
