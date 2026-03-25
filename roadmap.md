# FinCore roadmap (repo source of truth)

This file is the **canonical, repo-local** specification of what we are building and how we will deliver it in phases.

It serves two purposes:

- **Build spec**: the target architecture, services, security posture, and non-functional requirements (NFRs).
- **Execution plan**: milestone-based delivery where each milestone stays green (tests + static analysis).

## 0. System context

FinCore is a greenfield core banking and real-time payments platform. The system must prioritize:

- correctness (financial invariants)
- security (Zero Trust)
- auditability (event trails)
- reliability (high uptime)

## 1. Architecture mandates (non-negotiable)

### 1.1 Hexagonal Architecture (Ports & Adapters)

- **Domain** is pure Go (no framework or transport concerns).
- **Application** defines use cases and ports (interfaces).
- **Infrastructure** provides adapters (DB, gRPC, HTTP, Kafka).
- Dependencies always point inward.

### 1.2 CQRS

- Writes are commands.
- Reads are queries / projections.
- Command and query models are distinct.

### 1.3 Event Sourcing + Snapshotting

- Every state mutation appends an immutable event.
- Aggregates rebuild via replay.
- Snapshots taken every 50 events (configurable).

### 1.4 Saga pattern

- Choreography for simple flows.
- Orchestration (Temporal) for complex multi-step payments with compensation.

### 1.5 Transactional outbox

- Every domain change that emits integration events writes to an outbox table in the same DB transaction.
- A relay process publishes and marks delivered.

### 1.6 Resilience patterns

- Circuit breaker for downstream calls.
- Bulkheads (resource isolation) to prevent cascading failures.

## 2. Security and compliance targets

### 2.1 Compliance

- PCI-DSS Level 1
- SOC2 Type II
- GDPR

### 2.2 Security posture (Zero Trust)

- Service-to-service authentication via SPIFFE/SPIRE mTLS.
- JWT verification is performed **locally** using JWKS.
- Secrets must not be logged.
- Redirect URI validation is strict.

Repo invariants (enforced by code + tests):

- **JWKS canonical endpoint**: `/.well-known/jwks.json`
- **OAuth2 endpoints**:
  - `/oauth/authorize`
  - `/oauth/token`
- **Admin endpoints** must remain private behind the gateway.
- **Redirect URIs with fragments (`#...`) are rejected**.

## 3. Non-functional requirements (NFRs)

- Latency p99: < 50ms for payment authorization (planned; enforce with load tests later)
- Throughput: 10,000 TPS sustained, 50,000 TPS burst (planned)
- Availability: 99.999% (planned)
- RPO: < 1 second (planned)
- RTO: < 30 seconds (planned)
- Idempotency: 100% for payment APIs (planned)

## 4. Service catalog (target: 12 services)

This is the full target system. Services are deployed independently and communicate via gRPC (internal) and the gateway (external).

Service catalog conventions:

- Each service has a clearly defined **data ownership boundary**.
- Public APIs are exposed via `api-gateway` only.
- Internal service-to-service calls use SPIFFE/SPIRE mTLS.
- Integration events use the transactional outbox pattern.

- `api-gateway`
  - Public/private routing and policy enforcement
  - Rate limiting and request shaping by path
  - JWT verification via JWKS
  - Ownership:
    - Edge policy: routing, authN/authZ enforcement, and request limits
  - External interfaces:
    - HTTP JSON via gRPC-gateway for public endpoints
  - Key dependencies:
    - JWKS from `identity-service`
- `identity-service`
  - OIDC discovery and JWKS serving
  - OAuth2 authorization UX (`/oauth/authorize`) and token issuance (`/oauth/token`)
  - Client registry, PKCE enforcement, refresh tokens, sessions/consent
  - Ownership:
    - Identity, authentication, authorization tokens, OAuth2/OIDC
    - Client registry and redirect URI/scope allowlists
  - External interfaces:
    - `/.well-known/openid-configuration`
    - `/.well-known/jwks.json`
    - `/oauth/authorize`
    - `/oauth/token`
  - Data:
    - OAuth clients, authorization codes, refresh sessions, consent records
- `account-service`
  - Account lifecycle and account-level invariants
  - CQRS + Event Sourcing + snapshots
  - Ownership:
    - Customer accounts and account state machine
  - External interfaces:
    - Account open/close, status, balances (via gateway)
  - Data:
    - Event stream + snapshots, projections (read models)
- `ledger-service`
  - Double-entry ledger and posting rules
  - CQRS + Event Sourcing
  - Ownership:
    - Ledger entries, posting rules, and accounting correctness
  - External interfaces:
    - Post transactions, query balances and ledger lines (via gateway)
  - Data:
    - Event stream, immutable ledger records, projections
- `payment-service`
  - Payment initiation/authorization/settlement
  - Saga orchestration (Temporal) with compensations
  - Idempotency enforcement
  - Ownership:
    - Payment state machine, orchestration, and idempotency keys
  - External interfaces:
    - Initiate payment, authorize, settle, cancel, status (via gateway)
  - Key dependencies:
    - `account-service` for account validation
    - `ledger-service` for posting
    - `fraud-engine` for risk checks
    - `fx-service` for FX quotes (when applicable)
  - Data:
    - Payment aggregate events, idempotency records, workflow state references
- `fraud-engine`
  - Rules + ML scoring
  - Decisioning APIs for payment risk checks
  - Ownership:
    - Fraud rules, feature computation, and decisions
  - External interfaces:
    - Internal decisioning API (called by `payment-service`)
  - Data:
    - Rulesets, model versions, decision logs
- `vault-service`
  - PCI tokenization boundary
  - Sensitive data vaulting and cryptographic operations
  - Ownership:
    - Tokenization, encryption, and storage of sensitive identifiers
  - External interfaces:
    - Internal tokenization APIs
  - Data:
    - Vaulted records and cryptographic metadata
- `fx-service`
  - Multi-currency conversion and rate sourcing
  - FX quotation and settlement support
  - Ownership:
    - FX rate ingestion, quoting, and conversion logic
  - External interfaces:
    - Quote and conversion APIs (primarily internal)
  - Data:
    - FX rates, quotes, and settlement references
- `notification-service`
  - Event-driven customer/operator notifications
  - Templates and delivery channels (email/SMS/push)
  - Ownership:
    - Notification templates and delivery policy
  - External interfaces:
    - Internal send APIs and event subscribers
  - Data:
    - Templates, delivery logs, user preferences
- `reporting-service`
  - Read models and regulatory reports
  - Batch exports and analytics feeds
  - Ownership:
    - Read-optimized views and regulatory outputs
  - External interfaces:
    - Report generation and export APIs
  - Data:
    - OLAP projections (ClickHouse) and export artifacts
- `audit-service`
  - Tamper-evident audit trails
  - Compliance exports and retention policies
  - Ownership:
    - Compliance-grade audit trails and retrieval
  - External interfaces:
    - Audit query/export APIs (private by default)
  - Data:
    - Append-only audit log and export manifests
- `admin-service`
  - Internal-only operator tooling
  - Client/admin workflows (disable client, rotate secrets, support operations)
  - Ownership:
    - Internal admin workflows and operator approvals
  - External interfaces:
    - Private admin APIs only (never public)
  - Key dependencies:
    - `identity-service` client lifecycle endpoints
    - `audit-service` for audit capture and exports

## 5. Phase plan (milestone execution)

### Phase 0: repo baseline

- CI gates:
  - `gofmt` check
  - `go vet ./...`
  - `go test ./...`
  - `go test -race ./...`
  - `staticcheck ./...`
  - coverage threshold job

### Phase 1: foundation

- CQRS + Event Sourcing adoption for:
  - account-service
  - ledger-service
- Snapshotting (every 50 events)
- Transactional outbox pattern (shared package + service usage)

Acceptance criteria:

- Account open/deposit/withdraw behaviors are event-sourced.
- Ledger behaviors remain double-entry safe.
- Unit tests cover key invariants.

### Phase 2: identity and OAuth2

#### 2.1 OIDC/JWKS baseline

- OIDC discovery: `/.well-known/openid-configuration`
- JWKS endpoint: `/.well-known/jwks.json`
- Local JWT verification via JWKS

#### 2.2 OAuth2 Authorization Code + PKCE

- `/oauth/authorize` and `/oauth/token`
- PKCE S256 only
- Client registry
  - public/confidential clients
  - redirect URI allowlist
  - scope allowlist
- Confidential client auth via HTTP Basic at token endpoint
- OAuth2 error redirects for browser flow (`error`, `error_description`, `state`)
- CSRF protection on HTML authorize flow

Acceptance criteria:

- Full PKCE flow works end-to-end in local dev (documented in `docs/oauth-pkce-local-dev.md`).
- Redirect URI fragment rejection enforced.

#### 2.3 Next identity hardening milestones

- Token endpoint strictness (content-type, required params, RFC-aligned errors)
- Consent persistence and consent history
- Session store strategy (in-memory now; optional Redis later)
- Client lifecycle management
  - secret rotation
  - disable client
  - audit fields
- Observability for auth flows (metrics + tracing)

Acceptance criteria:

- OAuth token errors are RFC-aligned (`invalid_client`, `invalid_grant`, `invalid_request`) and include `WWW-Authenticate` where required.
- Consent can be granted/denied and is persisted and queryable for the client/user pair.
- Session fixation is prevented and cookies are hardened (secure, httpOnly, sameSite).
- Client lifecycle operations are auditable and do not weaken redirect URI validation.
- Traces and metrics exist for authorize/token endpoints with no secret leakage.

### Phase 3: payments and sagas

- Introduce `payment-service` as the saga coordinator (Temporal workflows)
- Introduce idempotency store (Redis)
- Integrate fraud-engine and FX

Acceptance criteria:

- A payment can be initiated and settled with compensations on failure.
- Idempotency prevents double-charging on retries.

Milestone deliverables:

- Payment state machine modeled as events + projections.
- Temporal workflows (orchestrator) with explicit compensation steps.
- Integration events published via outbox (payment initiated/authorized/settled/failed).

### Phase 4: reporting, audit, and observability

- Projections into OLAP store (ClickHouse)
- Audit service for compliance exports
- Golden signals
  - tracing (OpenTelemetry)
  - metrics (Prometheus)
  - logs (structured JSON)

Acceptance criteria:

- Every critical money movement emits an audit trail entry with immutable correlation IDs.
- Reporting projections are reproducible from source events.
- Dashboards exist for golden signals and error budgets.

### Phase 5: production readiness

- Kubernetes + Helm
- GitOps (ArgoCD)
- Terraform
- Chaos testing + load testing (k6)

Acceptance criteria:

- GitOps-managed environments exist for dev/stage/prod.
- mTLS identities are managed and rotated without downtime.
- Load tests and chaos experiments are in CI/CD pipelines with clear pass/fail gates.

## 6. Deliverables checklist (per phase)

For each delivered service/milestone:

- Protobuf definitions for gRPC interfaces
- Database migrations (where applicable)
- Unit tests (and integration tests where justified)
- Docs for:
  - public endpoints
  - environment variables
  - security behavior

## 7. Definition of done (for any milestone)

- Unit tests cover new behavior + edge cases
- No security boundary regressions (SPIFFE, JWKS verification, gateway routing)
- CI passes (same as `.github/workflows/ci.yml`):
  - `test -z "$(gofmt -l .)"`
  - `go vet ./...`
  - `go test ./...`
  - `go test -race ./...`
  - `staticcheck ./...`
  - coverage threshold job passes
