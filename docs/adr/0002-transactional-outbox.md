# ADR 0002: Transactional Outbox for reliable event publishing

## Status
Accepted

## Context
FinCore publishes domain events for integration (Kafka topics) and internal choreography. In a financial system, it is unacceptable to:

- persist a state change but fail to publish its event (silent integration failure)
- publish an event but fail to persist the state change (phantom event)

Directly writing to the database and separately publishing to Kafka creates a classic dual-write problem. Network failures, process crashes, or broker unavailability will produce inconsistencies.

## Decision
We will implement the Transactional Outbox pattern:

- In the same database transaction as event-store writes, the service inserts an outbox row in `outbox_messages`.
- A separate relay process polls `outbox_messages where published_at is null`, publishes to the message broker, and marks the row as published.

## Why this is production-grade
- Eliminates dual-write inconsistency: DB write and outbox enqueue are atomic.
- Supports retries: relay can retry publishing safely.
- Works with broker downtime: backlog accumulates in DB.
- Enables operational controls: DLQ routing, rate limiting, replay tools.

## Trade-offs
- Requires an additional worker (relay) and monitoring.
- Outbox table must be sized and maintained (retention, archiving).
- Publishing latency is bounded by poll interval (tunable).

## Consequences
- Write-side transactions must include outbox enqueue.
- Relay must use `FOR UPDATE SKIP LOCKED` (or equivalent) to scale horizontally without double-publish.
- We will add alerts on outbox backlog size and publish error rates.
