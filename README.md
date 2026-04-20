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

### Prerequisites
- **Docker & Docker Compose**
- **Go 1.23+**
- **Node.js 24+**

### Deployment in 3 Steps

1. **Spin up Infrastructure Cells**:
   ```bash
   docker-compose up -d
   ```
   *This starts CockroachDB, Kafka, Temporal, HashiCorp Vault, and SPIRE.*

2. **Initialize Secrets**:
   ```bash
   # (Scripts provided in /scripts to auto-unseal Vault)
   ./scripts/bootstrap-vault.sh
   ```

3. **Launch Dashboard**:
   ```bash
   cd webapp && npm install && npm run dev
   ```

## 📜 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

## 🤝 Contributing

We welcome contributions! Please see our [Security Guidelines](AGENTS.md) and [Roadmap](roadmap.md) before submitting a Pull Request.
