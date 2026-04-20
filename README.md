# FinCore OS: High-Performance Enterprise Banking Engine

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/rowjay007/fincore-mevi)](https://goreportcard.com/report/github.com/rowjay007/fincore-mevi)
[![Node Version](https://img.shields.io/badge/node-v24+-blue.svg)](https://nodejs.org)
[![Platform](https://img.shields.io/badge/platform-Cloud--Native-blueviolet.svg)](https://kubernetes.io)

FinCore is a high-performance, multi-region banking engine built to handle **10M+ daily transactions**. It implements a **Triple-Entry Immutable Ledger** with strong consistency across geo-distributed regions, designed for global financial institutions.

## 🏗️ Core Architecture

FinCore is built on a **Cell-Based Architecture**, ensuring that failures in one regional shard do not compromise the global ledger.

### 1. The Distributed Ledger (Triple-Entry)
Unlike traditional double-entry systems, FinCore uses a Triple-Entry approach:
- **Debit Entry**: Originating account movement.
- **Credit Entry**: Destination account movement.
- **Audit Receipt**: An immutable, Merkle-signed receipt stored in a separate Audit Chain.

### 2. Service Grid (12 Microservices)
The system is composed of specialized Go microservices communicating via **gRPC** and **mTLS**:
- **api-gateway**: Entry point with rate-limiting, JWT validation, and **Chaos Monkey** resilience middleware.
- **identity-service**: OIDC discovery + JWKS provider + **WebAuthn/Passkey** biometric support.
- **ledger-service**: Double-entry ledger with Merkle Tree integrity validation.
- **payment-service**: Distributed transaction orchestrator using the **Saga Pattern** via Temporal.
- **fraud-engine**: Real-time heuristic evaluation with high-precision latency.
- **vault-service**: Hardware-secured PII tokenization via **HashiCorp Vault**.

## 🚀 Key Enterprise Features

### 📡 High-Fidelity Observability
The Next.js dashboard provides a real-time command center:
- **Live Trace Explorer**: Visualize distributed tracing (OpenTelemetry) across the service grid.
- **3D Global Map**: Real-time liquidity movement monitoring across US, EU, and Asia cells.
- **AI Investigator**: Natural language behavioral forensics powered by Vercel AI SDK.

### 🛡️ Zero-Trust Security
- **SPIFFE/mTLS**: Automatic workload identities (SVIDs) via **SPIRE**. Inter-service traffic is impossible without valid cryptographic identities.
- **Cryptographic Audit**: Every transaction is hashed into a Merkle Tree. Any tampering with historical records is detected instantly.
- **Passkey Support**: Modern, passwordless authentication using biometric hardware (FIDO2/WebAuthn).

## ⚡ Quick Start

### 1. Prerequisites
- **Docker & Docker Compose**
- **Go 1.23+**
- **Node.js 24+** (for High-Fidelity Dashboard)

### 2. Environment Configuration
Copy the sample environment file and adjust variables as needed:
```bash
cp .env.example .env
```

### 3. Launching the Ecosystem

#### Phase A: Infrastructure & Security
Start the "Security Cell" (Vault & SPIRE) and core backing services:
```bash
# Start Vault & SPIRE
make sec-up

# Seed security identities (OIDC clients & SVIDs)
make sec-seed

# Start CockroachDB, Kafka, and Temporal
docker-compose up -d
```

#### Phase B: Dashboard
The command center provides real-time telemetry and management:
```bash
cd webapp
npm install
npm run dev
```

#### Phase C: Microservices
Each service can be started individually or via the service grid. Example:
```bash
# Start the Gateway (Entry point)
go run services/api-gateway/cmd/api-gateway/main.go

# Start Identity (Auth & WebAuthn)
go run services/identity-service/cmd/identity-service/*.go
```

## 🛠️ Service Map & Ports

| Service | Port | Description |
| :--- | :--- | :--- |
| **Dashboard** | `3000` | High-fidelity Next.js command center |
| **API Gateway** | `8080` | Entry point, Chaos Monkey, Rate Limiting |
| **Identity** | `8084` | OIDC, JWKS, WebAuthn/Passkey Login |
| **Vault** | `8200` | PII Tokenization & Secret Management |
| **CockroachDB** | `26257` | Distributed strong-consistency ledger |
| **Temporal** | `7233` | Distributed Saga Orchestration |

## 🧪 Testing & Resilience
Run the full suite to verify mTLS boundaries and Merkle integrity:
```bash
# Run all Go tests with race detection
go test -race ./...

# Trigger Chaos Monkey in Gateway
export ENABLE_CHAOS=true
```

## 📜 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

We welcome contributions! Please see our [Security Guidelines](AGENTS.md) and [Roadmap](roadmap.md) before submitting a Pull Request.
