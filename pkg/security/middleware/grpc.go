package middleware

import (
	"context"
	"strings"

	"fincore/pkg/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryAuthzInterceptor(tokens security.TokenMaker, suffixToRequiredPerm map[string]string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requiresAuth := false
		requiredPerm := ""
		for suffix, perm := range suffixToRequiredPerm {
			if strings.HasSuffix(info.FullMethod, suffix) {
				requiresAuth = true
				requiredPerm = perm
				break
			}
		}
		if !requiresAuth {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, security.ErrInvalidToken
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			return nil, security.ErrInvalidToken
		}
		v := strings.TrimSpace(vals[0])
		const prefix = "Bearer "
		if !strings.HasPrefix(v, prefix) {
			return nil, security.ErrInvalidToken
		}
		tok := strings.TrimSpace(strings.TrimPrefix(v, prefix))
		if tok == "" {
			return nil, security.ErrInvalidToken
		}

		payload, err := tokens.VerifyToken(tok)
		if err != nil {
			return nil, err
		}

		if requiredPerm != "" {
			allowed := false
			for _, p := range payload.Permissions {
				if p == requiredPerm {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, security.ErrInvalidToken
			}
		}

		return handler(ctx, req)
	}
}
