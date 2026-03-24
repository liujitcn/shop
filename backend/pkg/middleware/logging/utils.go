package logging

import (
	"net"
	"net/http"
	"strings"

	"github.com/liujitcn/go-utils/geoip/qqwry"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

var ipClient = qqwry.NewClient()

// extractAuthToken 从JWT Token中提取用户信息
func extractAuthToken(authToken string, authenticator authnEngine.Authenticator) *authData.UserTokenPayload {
	if len(authToken) == 0 {
		return nil
	}

	jwtToken := strings.TrimPrefix(authToken, "Bearer ")
	authnClaims, _ := authenticator.AuthenticateToken(jwtToken)
	if authnClaims == nil {
		return nil
	}

	ut, _ := authData.NewUserTokenPayloadWithClaims(authnClaims)
	if ut == nil {
		return nil
	}

	return ut
}

// getClientRealIP 获取客户端真实IP
func getClientRealIP(request *http.Request) string {
	if request == nil {
		return ""
	}

	// 先检查 X-Forwarded-For 头
	// 由于它可以记录整个代理链中的IP地址，因此适用于多级代理的情况。
	// 当请求经过多个代理服务器时，X-Forwarded-For字段可以完整地记录原始请求的客户端IP地址和所有代理服务器的IP地址。
	// 需要注意：
	// 最外层Nginx配置为：proxy_set_header X-Forwarded-For $remote_addr; 如此做可以覆写掉ip。以防止ip伪造。
	// 里层Nginx配置为：proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
	xff := request.Header.Get(HeaderKeyXForwardedFor)
	if xff != "" {
		// X-Forwarded-For字段的值是一个逗号分隔的IP地址列表，
		// 一般来说，第一个IP地址是原始请求的客户端IP地址（当然，它可以被伪造）。
		ips := strings.Split(xff, ",")

		for _, ip := range ips {
			// 去除空格
			ip = strings.TrimSpace(ip)
			// 检查是否是合法的IP地址
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// 接着检查反向代理的 X-Real-IP 头
	// 通常只在反向代理服务器中使用，并且只记录原始请求的客户端IP地址。
	// 它不适用于多级代理的情况，因为每经过一个代理服务器，X-Real-IP字段的值都会被覆盖为最新的客户端IP地址。
	xri := request.Header.Get(HeaderKeyXRealIP)
	if xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	return getIPFromRemoteAddr(request.RemoteAddr)
}

func getIPFromRemoteAddr(hostAddress string) string {
	// Check if the host address contains a port
	if strings.Contains(hostAddress, ":") {
		// Attempt to split the host address into host and port
		host, _, err := net.SplitHostPort(strings.TrimSpace(hostAddress))
		if err == nil {
			// Validate the host as an IP address
			if net.ParseIP(host) != nil {
				return host
			}
		}
	}
	// Validate the host address as an IP address
	if net.ParseIP(hostAddress) != nil {
		return hostAddress
	}
	return ""
}

// getRequestId 获取请求ID
func getRequestId(request *http.Request) string {
	if request == nil {
		return ""
	}

	// 先检查 X-Request-ID 头
	// 这是比较常见的用于标识请求的自定义头部字段。
	// 例如，在一个微服务架构的系统中，当一个请求从前端应用发送到后端的多个微服务时，
	// 每个微服务都可以在 X-Request-ID 字段中获取到相同的请求标识，从而方便追踪请求在各个服务节点中的处理情况。
	xri := request.Header.Get(HeaderKeyXRequestID)
	if xri != "" {
		return xri
	}

	// 接着检查 X-Correlation-ID 头
	// 它和 X-Request-ID 类似，用于关联一系列相关的请求或者事务。
	// 比如，在一个包含多个子请求的复杂业务流程中，X-Correlation-ID 可以用于跟踪整个业务流程中各个子请求之间的关系。
	xci := request.Header.Get(HeaderKeyXCorrelationID)
	if xci != "" {
		return xci
	}

	// 函数计算的请求ID
	xfcri := request.Header.Get(HeaderKeyXFcRequestID)
	if xfcri != "" {
		return xfcri
	}

	return ""
}

// clientIpToLocation 获取客户端IP的地理位置
func clientIpToLocation(ip string) string {
	res, err := ipClient.Query(ip)
	if err != nil {
		return ""
	}
	return res.City
}
