# ADR 0001: Event Sourcing for write model

## Status
Accepted

## Context
FinCore is a core banking and real-time payments platform where correctness, auditability, and forensic replay are non-negotiable. We also operate under stringent compliance expectations (auditability, change traceability, tamper evidence, and incident reconstruction).

Traditional “state as rows” persistence makes it hard to prove *how* a balance or account state reached its current value, and it is easy to accidentally lose historical detail (e.g., updates that overwrite previous values).

## Decision
We will persist all state changes as an append-only stream of immutable events in an event store.

- Each aggregate has an ordered event stream identified by `aggregate_id`.
- Aggregate state is rebuilt by replaying events.
- Every mutation is expressed as a domain event.

## Why this is a production choice
- Audit trail is intrinsic: every change is captured as an immutable record.
- Debuggability/forensics: we can replay a customer’s timeline or reproduce incidents.
- Integration: domain events become the system’s integration contract (Kafka topics) without dual-writing.
- Enables strong invariants in the domain: state transitions happen only via domain methods.

## Trade-offs
- More moving parts: event schema/versioning and replay logic.
- Querying current state requires projections or rehydration.
- Requires operational maturity (event store backups, retention, schema evolution discipline).

## Consequences
- We implement an append-only `event_store_events` table with optimistic concurrency (unique `(aggregate_id, version)`).
- We add snapshots (ADR 0004) to keep replay cost bounded.
- Read models are projections (CQRS) and are explicitly treated as derived data.
