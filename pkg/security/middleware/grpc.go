package middleware

import (
	"context"
	"strings"

	"fincore/pkg/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
			return nil, status.Error(codes.Unauthenticated, "missing auth metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			vals = md.Get("Authorization")
		}
		if len(vals) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization")
		}
		v := strings.TrimSpace(vals[0])
		const prefix = "Bearer "
		if !strings.HasPrefix(v, prefix) {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}
		tok := strings.TrimSpace(strings.TrimPrefix(v, prefix))
		if tok == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization")
		}

		payload, err := tokens.VerifyToken(tok)
		if err != nil {
			if err == security.ErrExpiredToken || err == security.ErrInvalidToken {
				return nil, status.Error(codes.Unauthenticated, "invalid token")
			}
			return nil, status.Error(codes.Unauthenticated, "invalid token")
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
				return nil, status.Error(codes.PermissionDenied, "forbidden")
			}
		}

		return handler(ctx, req)
	}
}
