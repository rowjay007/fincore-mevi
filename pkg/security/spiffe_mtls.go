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

func NewSpiffeMTLSClientCredentials(ctx context.Context) (credentials.TransportCredentials, func(), error) {
	if !spiffeMTLSEnabled() {
		return nil, func() {}, errors.New("spiffe mtls not enabled")
	}

	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(spiffeWorkloadAPIAddr())))
	if err != nil {
		return nil, func() {}, err
	}

	trustDomain, err := spiffeTrustDomain()
	if err != nil {
		source.Close()
		return nil, func() {}, err
	}
	authorizer := tlsconfig.AuthorizeMemberOf(trustDomain)
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

	trustDomain, err := spiffeTrustDomain()
	if err != nil {
		source.Close()
		return nil, func() {}, err
	}
	authorizer := tlsconfig.AuthorizeMemberOf(trustDomain)
	tlsCfg := tlsconfig.MTLSServerConfig(source, source, authorizer)
	tlsCfg.MinVersion = tls.VersionTLS12

	creds := credentials.NewTLS(tlsCfg)
	return creds, func() { source.Close() }, nil
}
