# fincore-mevi

## Identity + JWKS

The platform uses JWTs signed by the internal `identity-service`. Other services verify tokens locally using the identity JWKS endpoint.

Canonical JWKS URL:

```bash
export AUTH_JWKS_URL=http://localhost:8084/.well-known/jwks.json
```

## SPIFFE/SPIRE mTLS

### Reason (why)

SPIFFE/SPIRE provides workload identities (SVIDs) so services can authenticate each other using mTLS without distributing long-lived certificates. This supports a Zero Trust posture and prepares the platform for Istio mTLS in Kubernetes.

### Usage (local dev)

See `docs/spiffe-spire-local-dev.md`.

To enable in-process gRPC mTLS (opt-in):

```bash
export SPIFFE_MTLS_ENABLED=true
export SPIFFE_TRUST_DOMAIN=fincore.local
export SPIFFE_WORKLOAD_API_ADDR=unix:///run/spire/sockets/agent.sock
```

Optional allowlists (least-privilege):

```bash
export SPIFFE_MTLS_CLIENT_ALLOWED_SVIDS=spiffe://fincore.local/ns/default/sa/ledger-service
export SPIFFE_MTLS_SERVER_ALLOWED_SVIDS=spiffe://fincore.local/ns/default/sa/account-service
```
