package middleware

import (
	"context"
	"crypto/x509"
	"os"
	"strings"

	"fincore/pkg/security"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func gatewaySpiffeIDFromEnv() string {
	v := strings.TrimSpace(os.Getenv("GATEWAY_SPIFFE_ID"))
	if v == "" {
		v = "spiffe://fincore.local/ns/default/sa/api-gateway"
	}
	return v
}

func peerSpiffeIDFromContext(ctx context.Context) (spiffeid.ID, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok || p == nil {
		return spiffeid.ID{}, false
	}

	ai := p.AuthInfo
	if ai == nil {
		return spiffeid.ID{}, false
	}

	tlsInfo, ok := ai.(credentials.TLSInfo)
	if !ok {
		return spiffeid.ID{}, false
	}
	state := tlsInfo.State
	if state.Version == 0 {
		return spiffeid.ID{}, false
	}
	if len(state.PeerCertificates) == 0 {
		return spiffeid.ID{}, false
	}
	leaf := state.PeerCertificates[0]
	return spiffeIDFromLeafCert(leaf)
}

func spiffeIDFromLeafCert(leaf *x509.Certificate) (spiffeid.ID, bool) {
	if leaf == nil {
		return spiffeid.ID{}, false
	}
	for _, uri := range leaf.URIs {
		if uri == nil {
			continue
		}
		id, err := spiffeid.FromURI(uri)
		if err == nil {
			return id, true
		}
	}
	return spiffeid.ID{}, false
}

func payloadFromGatewayHeaders(ctx context.Context) (*security.TokenPayload, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, false
	}

	subject := strings.TrimSpace(firstMDValue(md, "x-fincore-subject", "X-Fincore-Subject"))
	if subject == "" {
		return nil, false
	}

	peerID, ok := peerSpiffeIDFromContext(ctx)
	if !ok {
		return nil, false
	}
	allowed, err := spiffeid.FromString(gatewaySpiffeIDFromEnv())
	if err != nil {
		return nil, false
	}
	if peerID != allowed {
		return nil, false
	}

	roles := splitCSV(firstMDValue(md, "x-fincore-roles", "X-Fincore-Roles"))
	perms := splitCSV(firstMDValue(md, "x-fincore-permissions", "X-Fincore-Permissions"))

	return &security.TokenPayload{UserID: subject, Roles: roles, Permissions: perms}, true
}

func firstMDValue(md metadata.MD, keys ...string) string {
	for _, k := range keys {
		vals := md.Get(k)
		if len(vals) != 0 {
			return vals[0]
		}
	}
	return ""
}

func splitCSV(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

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

		var payload *security.TokenPayload
		if len(vals) == 0 {
			// Support gateway-asserted identity headers only when the gateway is the mTLS peer.
			if p, ok := payloadFromGatewayHeaders(ctx); ok {
				payload = p
			} else {
				return nil, status.Error(codes.Unauthenticated, "missing authorization")
			}
		} else {
			v := strings.TrimSpace(vals[0])
			const prefix = "Bearer "
			if !strings.HasPrefix(v, prefix) {
				return nil, status.Error(codes.Unauthenticated, "invalid authorization")
			}
			tok := strings.TrimSpace(strings.TrimPrefix(v, prefix))
			if tok == "" {
				return nil, status.Error(codes.Unauthenticated, "invalid authorization")
			}

			p, err := tokens.VerifyToken(tok)
			if err != nil {
				if err == security.ErrExpiredToken || err == security.ErrInvalidToken {
					return nil, status.Error(codes.Unauthenticated, "invalid token")
				}
				return nil, status.Error(codes.Unauthenticated, "invalid token")
			}
			payload = p
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
