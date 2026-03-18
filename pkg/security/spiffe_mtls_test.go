package security

import (
	"os"
	"testing"
)

func TestSpiffeMTLSEnabled(t *testing.T) {
	t.Setenv("SPIFFE_MTLS_ENABLED", "")
	if spiffeMTLSEnabled() {
		t.Fatalf("expected disabled when env is empty")
	}

	for _, v := range []string{"1", "true", "yes", "on", "TRUE", " YeS "} {
		t.Setenv("SPIFFE_MTLS_ENABLED", v)
		if !spiffeMTLSEnabled() {
			t.Fatalf("expected enabled for %q", v)
		}
	}

	t.Setenv("SPIFFE_MTLS_ENABLED", "0")
	if spiffeMTLSEnabled() {
		t.Fatalf("expected disabled for 0")
	}
}

func TestSpiffeWorkloadAPIAddr_DefaultAndOverride(t *testing.T) {
	t.Setenv("SPIFFE_WORKLOAD_API_ADDR", "")
	if got := spiffeWorkloadAPIAddr(); got != "unix:///run/spire/sockets/agent.sock" {
		t.Fatalf("unexpected default workload api addr: %q", got)
	}

	t.Setenv("SPIFFE_WORKLOAD_API_ADDR", "unix:///tmp/workload.sock")
	if got := spiffeWorkloadAPIAddr(); got != "unix:///tmp/workload.sock" {
		t.Fatalf("unexpected overridden workload api addr: %q", got)
	}
}

func TestSpiffeTrustDomain_DefaultAndInvalid(t *testing.T) {
	t.Setenv("SPIFFE_TRUST_DOMAIN", "")
	if _, err := spiffeTrustDomain(); err != nil {
		t.Fatalf("expected default trust domain to parse: %v", err)
	}

	t.Setenv("SPIFFE_TRUST_DOMAIN", "not a domain")
	if _, err := spiffeTrustDomain(); err == nil {
		t.Fatalf("expected invalid trust domain to error")
	}
}

func TestSpiffeAllowedIDsFromEnv(t *testing.T) {
	key := "SPIFFE_TEST_ALLOWED"
	_ = os.Unsetenv(key)
	ids, ok, err := spiffeAllowedIDsFromEnv(key)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when unset")
	}
	if ids != nil {
		t.Fatalf("expected nil ids when unset")
	}

	t.Setenv(key, " , ")
	ids, ok, err = spiffeAllowedIDsFromEnv(key)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false when only commas/spaces")
	}
	if ids != nil {
		t.Fatalf("expected nil ids when empty")
	}

	t.Setenv(key, "spiffe://fincore.local/ns/default/sa/account-service")
	ids, ok, err = spiffeAllowedIDsFromEnv(key)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok || len(ids) != 1 {
		t.Fatalf("expected 1 id, ok=true; got ok=%v len=%d", ok, len(ids))
	}

	t.Setenv(key, "spiffe://fincore.local/ns/default/sa/a,not-a-spiffe-id")
	_, _, err = spiffeAllowedIDsFromEnv(key)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestSpiffeAuthorizers_SelectAllowlistVsTrustDomain(t *testing.T) {
	// When allowlists are set, authorizer creation should succeed without needing a valid trust domain.
	t.Setenv("SPIFFE_TRUST_DOMAIN", "not a domain")
	t.Setenv("SPIFFE_MTLS_CLIENT_ALLOWED_SVIDS", "spiffe://fincore.local/ns/default/sa/ledger-service")
	t.Setenv("SPIFFE_MTLS_SERVER_ALLOWED_SVIDS", "spiffe://fincore.local/ns/default/sa/account-service")

	if _, err := spiffeClientAuthorizer(); err != nil {
		t.Fatalf("expected client authorizer to succeed with allowlist: %v", err)
	}
	if _, err := spiffeServerAuthorizer(); err != nil {
		t.Fatalf("expected server authorizer to succeed with allowlist: %v", err)
	}

	// Without allowlists, invalid trust domain should error.
	t.Setenv("SPIFFE_MTLS_CLIENT_ALLOWED_SVIDS", "")
	t.Setenv("SPIFFE_MTLS_SERVER_ALLOWED_SVIDS", "")
	if _, err := spiffeClientAuthorizer(); err == nil {
		t.Fatalf("expected client authorizer to fail when trust domain invalid and no allowlist")
	}
	if _, err := spiffeServerAuthorizer(); err == nil {
		t.Fatalf("expected server authorizer to fail when trust domain invalid and no allowlist")
	}
}
