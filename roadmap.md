# fincore-mevi roadmap

This document is the in-repo guide for building the FinCore platform in small, verifiable milestones.

## Guiding principles

- Keep services internally authenticated (SPIFFE mTLS where enabled).
- Prefer local verification (JWKS) over network introspection.
- Event-sourcing + CQRS is the default for domain services.
- Reliable integration events via transactional outbox.
- Every milestone must include:
  - tests
  - docs updates (as needed)
  - `go test ./...` + `staticcheck ./...` passing

## Phase 0: repo baseline

- Service scaffolds and local dev docs
- CI: fmt/vet/test/race/staticcheck + coverage threshold for security packages

## Phase 1: foundation (completed)

- Account + ledger refactor to CQRS + Event Sourcing
- Snapshotting support
- Transactional outbox for reliable event publishing
- Local dev posture for SPIFFE/SPIRE and Vault (opt-in)

## Phase 2: identity (OIDC provider) + OAuth2 (in progress)

### 2.1 OIDC/JWKS baseline (completed)

- `identity-service` issues JWTs
- Canonical JWKS endpoint:
  - `/.well-known/jwks.json`
- Discovery endpoint:
  - `/.well-known/openid-configuration`

### 2.2 OAuth2 Authorization Code + PKCE (completed)

- Public endpoints:
  - `/oauth/authorize`
  - `/oauth/token`
- PKCE:
  - S256 only
- Persistent OAuth client registry
  - public/confidential clients
  - redirect URI allowlist
  - scope allowlist
- Token exchange supports confidential client authentication (HTTP Basic)
- Browser-friendly `/oauth/authorize` experience (HTML + lightweight session cookie)
- OAuth2-style error redirects (`error`, `error_description`, `state`)

### 2.3 Next hardening milestones (next)

- Discovery completeness and correctness
  - advertise supported auth methods at token endpoint
  - document expected OAuth2 error behaviors
- Token endpoint strictness
  - ensure invalid client behavior is consistent (including `WWW-Authenticate` semantics where appropriate)
  - enforce content-type and parameter parsing edge cases
- Consent and session hardening
  - optional persistence for consent decisions
  - session store strategy (in-memory now; optional Redis later)
  - CSRF defenses for browser form posts
- Client lifecycle
  - rotate client secret
  - disable client
  - audit fields (created_by/created_at/rotated_at)
- Operational hardening
  - metrics/tracing hooks
  - structured logs
  - rate limit policy review

## Phase 3: platform security posture (planned)

- SPIFFE/SPIRE automation for local dev and dev clusters
- Vault integration patterns
  - KV v2 for DSNs/secrets (opt-in)
  - agent auto-auth + templating where appropriate
- Secret rotation playbooks

## Phase 4: service-to-service auth propagation (planned)

- Gateway-to-internal propagation rules (headers/metadata)
- Fine-grained authorization primitives (RBAC/ABAC)
- Admin surface minimization and private routing guarantees

## Phase 5: production readiness (planned)

- Observability
  - metrics (Prometheus)
  - tracing (OpenTelemetry)
  - log correlation
- Deployment
  - Docker images
  - Kubernetes manifests/Helm
  - staging/prod config profiles
- Load and chaos testing

## Definition of done (for any milestone)

- Unit tests cover new behavior and edge cases
- No security boundary regressions (SPIFFE, JWKS verification, gateway public/private routing)
- CI passes:
  - `gofmt` clean
  - `go vet ./...`
  - `go test ./...`
  - `go test -race ./...`
  - `staticcheck ./...`
  - coverage threshold job passes
