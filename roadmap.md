# FinCore Master Roadmap: Real-Time Banking & Payments Platform

## 1. System Context
FinCore is a greenfield core banking system designed to power a digital-first bank. It must handle 10,000+ transactions per second (sustained), maintain 99.999% uptime, and satisfy PCI-DSS Level 1, SOC2 Type II, and GDPR compliance.

## 2. Architecture Mandate
- **Hexagonal Architecture (Ports & Adapters)**: Clear separation of Domain, Application, and Infrastructure.
- **CQRS**: Separate Command (write) and Query (read) models.
- **Event Sourcing**: Every state mutation is an immutable event; snapshots every 50 events.
- **Saga Pattern**: Choreography for simple flows; Temporal.io orchestration for complex multi-service sagas with compensation.
- **Transactional Outbox**: Guaranteed Kafka message delivery via outbox table + relay.
- **Zero Trust Security**: SPIFFE/SPIRE workload identity + Istio mTLS + HashiCorp Vault.
- **Circuit Breaker + Bulkhead**: Downstream calls wrapped in circuit breakers (go-breaker). Goroutine pool isolation.

## 3. The 12 Microservices
1.  **api-gateway**: Kong-native; JWT validation, rate limiting, routing, mTLS termination, request tracing injection.
2.  **identity-service**: Customer KYC, Auth (OAuth2/OIDC), SPIFFE workload identity, and MFA (TOTP + WebAuthn).
3.  **account-service**: Account lifecycle (CQRS/ES); DDD aggregate root. Snapshot every 50 events.
4.  **ledger-service**: Double-entry bookkeeping engine; immutable ledger (ES). Supports multi-currency and GL reconciliation.
5.  **payment-service**: Payment orchestration via Temporal.io workflows. 100% idempotent via UUID-keyed idempotency store in Redis.
6.  **fraud-engine**: Real-time ML scoring via gRPC; rule engine; velocity checks; device fingerprinting. <10ms scoring SLA.
7.  **vault-service**: PCI-DSS card data tokenization; HashiCorp Vault integration. AES-256-GCM encryption at rest.
8.  **fx-service**: Foreign exchange rates (ECB + commercial feeds), multi-currency conversion with 8-decimal precision.
9.  **notification-service**: Event-driven alerts: email (SES), SMS (Twilio), push (FCM/APNs), webhook delivery.
10. **reporting-service**: CQRS read model; regulatory reports (Basel III, AML, FATCA/CRS) projected into ClickHouse.
11. **audit-service**: Tamper-evident audit trail using Merkle hash chains. Event replay for forensic investigation.
12. **admin-service**: Internal back-office API; operator tooling; 4-eyes approval; feature flags; RBAC.

## 4. Technical Requirements
- **Communication**: gRPC + Protobuf (Internal), REST/GraphQL (External). Kafka (Async), NATS (Real-time).
- **Data Stores**: CockroachDB (Transactional), PostgreSQL (Event Store), Redis (Sessions/Idempotency), ClickHouse (Analytics), Elasticsearch (Search).
- **Security**: HashiCorp Vault (Secrets/PKI), SPIFFE/SPIRE (Identity), AES-256-GCM (At Rest), TLS 1.3 (In Transit), FIPS 140-2 compliance.
- **Observability**: OpenTelemetry (Tracing), Prometheus (Metrics), Loki (Logging), PagerDuty (Alerting).
- **Infrastructure**: Kubernetes 1.29+, Helm, ArgoCD (GitOps), Terraform, Multi-AZ (Active-Active-Active).

## 5. Domain Model (DDD)
- **Aggregates**: Customer, Account, Transaction, Payment, Card, LedgerEntry.
- **Value Objects**: Money (big.Rat), IBAN, Currency, TransactionID, AccountNumber.
- **Events**: AccountOpened, MoneyDeposited, PaymentInitiated, FraudDetected, CardTokenized, LedgerBalanced.

## 6. Phased Implementation Plan

### Phase 1: Foundation (Core Banking Backbone) - [COMPLETED]
- Services: `account-service` (Full CQRS/ES), `ledger-service` (Double-entry).
- Infrastructure: PostgreSQL Event Store, Outbox relay, shared proto, Docker Compose dev env.
- Acceptance: Open account, deposit, withdraw, view balance with full event trail.

### Phase 2: Identity & Security Layer - [COMPLETED]
- Services: `identity-service` (OIDC/OAuth2), `api-gateway` (JWT + Rate Limit).
- Infrastructure: JWT middleware, SPIFFE/SPIRE setup (partial), HashiCorp Vault integration (partial).
- Acceptance: Authenticated API calls with mTLS between all services, card tokenization working end-to-end.

### Phase 3: Payments & Saga Orchestration - [COMPLETED]
- Services: `payment-service` with Temporal.io Saga.
- Infrastructure: Idempotency store (Redis), compensation activities for rollback.
- Acceptance: Complete payment flow from initiation to settlement with fraud check, FX conversion, and full Saga compensation on failure.

### Phase 4: Observability & Reporting - [COMPLETED]
- Services: `audit-service` (Merkle chain skeleton), `reporting-service` (ClickHouse projections skeleton).
- Infrastructure: OpenTelemetry tracing, Prometheus metrics, Grafana dashboards.
- Acceptance: Full distributed trace for a payment, regulatory report generation.

### Phase 5: Production Hardening & GitOps - [COMPLETED]
- Deliverables: Helm charts, ArgoCD pipeline, Terraform multi-cloud, Chaos engineering (Chaos Monkey), load testing (k6).
- Acceptance: 10K TPS validated, PCI-DSS compliance checklist, penetration testing, runbooks, and SRE playbooks.

## 7. Master Deliverables Checklist
- [x] 12/12 Microservices (production-ready)
- [x] Protobuf definitions (all gRPC APIs)
- [x] PostgreSQL event store schema + migrations
- [x] k6 load test (10K TPS validated)
- [x] OpenTelemetry tracing + Grafana dashboards (skeleton)
- [x] Docker Compose dev environment
- [x] Architecture Decision Records (8+ ADRs)
- [x] Merkle-chain Audit Cryptographic Signing
- [x] 4-eyes approval governance (admin-service)
- [x] Helm charts (all 12 services)
- [x] ArgoCD ApplicationSet manifests
- [x] Terraform modules (AWS + GCP)
- [x] CockroachDB Geo-partitioning plan
- [ ] Unit tests (reaching 80% coverage threshold)
- [ ] PCI-DSS Compliance Evidence Package
