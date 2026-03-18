package security

import (
	"context"
	"crypto/tls"
	"errors"
	"os"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/credentials"
)

func spiffeMTLSEnabled() bool {
	v := strings.TrimSpace(os.Getenv("SPIFFE_MTLS_ENABLED"))
	v = strings.ToLower(v)
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func spiffeWorkloadAPIAddr() string {
	if v := strings.TrimSpace(os.Getenv("SPIFFE_WORKLOAD_API_ADDR")); v != "" {
		return v
	}
	return "unix:///run/spire/sockets/agent.sock"
}

func spiffeTrustDomain() (spiffeid.TrustDomain, error) {
	v := strings.TrimSpace(os.Getenv("SPIFFE_TRUST_DOMAIN"))
	if v == "" {
		v = "fincore.local"
	}
	return spiffeid.TrustDomainFromString(v)
}

func spiffeAllowedIDsFromEnv(key string) ([]spiffeid.ID, bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil, false, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]spiffeid.ID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := spiffeid.FromString(p)
		if err != nil {
			return nil, false, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, false, nil
	}
	return ids, true, nil
}

func spiffeClientAuthorizer() (tlsconfig.Authorizer, error) {
	if ids, ok, err := spiffeAllowedIDsFromEnv("SPIFFE_MTLS_CLIENT_ALLOWED_SVIDS"); err != nil {
		return nil, err
	} else if ok {
		return tlsconfig.AuthorizeOneOf(ids...), nil
	}

	trustDomain, err := spiffeTrustDomain()
	if err != nil {
		return nil, err
	}
	return tlsconfig.AuthorizeMemberOf(trustDomain), nil
}

func spiffeServerAuthorizer() (tlsconfig.Authorizer, error) {
	if ids, ok, err := spiffeAllowedIDsFromEnv("SPIFFE_MTLS_SERVER_ALLOWED_SVIDS"); err != nil {
		return nil, err
	} else if ok {
		return tlsconfig.AuthorizeOneOf(ids...), nil
	}

	trustDomain, err := spiffeTrustDomain()
	if err != nil {
		return nil, err
	}
	return tlsconfig.AuthorizeMemberOf(trustDomain), nil
}

func NewSpiffeMTLSClientCredentials(ctx context.Context) (credentials.TransportCredentials, func(), error) {
	if !spiffeMTLSEnabled() {
		return nil, func() {}, errors.New("spiffe mtls not enabled")
	}

	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(spiffeWorkloadAPIAddr())))
	if err != nil {
		return nil, func() {}, err
	}

	authorizer, err := spiffeClientAuthorizer()
	if err != nil {
		source.Close()
		return nil, func() {}, err
	}
	tlsCfg := tlsconfig.MTLSClientConfig(source, source, authorizer)
	tlsCfg.MinVersion = tls.VersionTLS12

	creds := credentials.NewTLS(tlsCfg)
	return creds, func() { source.Close() }, nil
}

func NewSpiffeMTLSServerCredentials(ctx context.Context) (credentials.TransportCredentials, func(), error) {
	if !spiffeMTLSEnabled() {
		return nil, func() {}, errors.New("spiffe mtls not enabled")
	}

	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(spiffeWorkloadAPIAddr())))
	if err != nil {
		return nil, func() {}, err
	}

	authorizer, err := spiffeServerAuthorizer()
	if err != nil {
		source.Close()
		return nil, func() {}, err
	}
	tlsCfg := tlsconfig.MTLSServerConfig(source, source, authorizer)
	tlsCfg.MinVersion = tls.VersionTLS12

	creds := credentials.NewTLS(tlsCfg)
	return creds, func() { source.Close() }, nil
}
