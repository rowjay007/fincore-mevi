package middleware

import (
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func GatewayAuthHeaderForwarder() runtime.ServeMuxOption {
	return runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
		if strings.EqualFold(key, "Authorization") {
			return "authorization", true
		}
		return runtime.DefaultHeaderMatcher(key)
	})
}
