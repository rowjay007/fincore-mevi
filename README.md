# FinCore OS: Enterprise Banking Engine

FinCore is a high-performance, multi-region banking engine built for 10M+ daily transactions. It implements a **Triple-Entry Immutable Ledger** with strong consistency across geo-distributed regions.

## 🏗️ High-Level Architecture

The system is composed of 12 core microservices orchestrated via **Temporal** and synchronized via **Kafka**.

### Service Grid
- **api-gateway**: Entry point with rate-limiting, JWT validation, and Chaos Monkey middleware.
- **identity-service**: OIDC discovery + JWKS provider + WebAuthn/Passkey support.
- **auth-service**: Core OAuth2 PKCE logic & client registry.
- **account-service**: CQRS-based account lifecycle management.
- **ledger-service**: Double-entry ledger with Merkle Tree integrity validation.
- **payment-service**: Distributed transaction orchestrator (Saga pattern).
- **fraud-engine**: Real-time heuristic and ML-based risk evaluation.
- **audit-service**: Immutable audit logging and Merkle chain verification.
- **fx-service**: Real-time currency exchange and settlement.
- **vault-service**: PII tokenization and secret management via HashiCorp Vault.
- **notification-service**: Async event-driven alerts.
- **reporting-service**: OLAP data warehouse synchronization.

## 🚀 Key Features

### 📡 High-Fidelity Observability
The Next.js dashboard provides a real-time command center:
- **Live Trace Explorer**: Full gRPC span visualization via OpenTelemetry.
- **3D Global Map**: Real-time liquidity movement across US, EU, and Asia.
- **AI Investigator**: Natural language behavioral forensics powered by Vercel AI SDK.

### 🛡️ Zero-Trust Security
- **SPIFFE/mTLS**: Automated workload identities for all inter-service communication.
- **WebAuthn/Passkeys**: Biometric-first authentication.
- **Merkle Chain Integrity**: Cryptographically proven ledger history.

## 🛠️ Infrastructure Stack
- **Database**: CockroachDB (Strong Consistency, Geo-Partitioned).
- **Messaging**: Kafka (Event Sourcing & Outbox Relay).
- **Orchestration**: Temporal (Distributed Sagas & Workflows).
- **Secrets**: HashiCorp Vault.
- **Identity**: SPIRE (Workload API).

## ⚡ Quick Start

```bash
# 1. Start Infrastructure (Postgres, Kafka, Temporal, Vault, Spire)
docker-compose up -d

# 2. Start Dashboard
cd webapp && npm install && npm run dev

# 3. Enable Chaos Engineering (Optional)
export ENABLE_CHAOS=true
```

## 📖 Documentation
- [Security Guidelines](AGENTS.md)
- [SPIFFE/SPIRE Local Dev](docs/spiffe-spire-local-dev.md)
- [Roadmap](roadmap.md)
