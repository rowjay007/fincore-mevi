# Phase 1 Load Tests (k6)

## Prereqs

- Install `k6`.
- Run the services (HTTP gateways):
  - auth-service (default `http://localhost:8082`)
  - account-service (default `http://localhost:8080`)
  - ledger-service (default `http://localhost:8083`)

## Auth

The gateways are auth-protected. Provide either:

- `K6_ACCESS_TOKEN` (preferred): a JWT access token that has permissions:
  - `account:write`, `account:read`
  - `ledger:read`

OR

- `K6_AUTH_EMAIL` + `K6_AUTH_PASSWORD`: credentials for a user that already has those permissions.

## Run

```bash
k6 run tests/load/k6_phase1_flow.js
```

## Env vars

- `AUTH_SERVICE_URL` (default `http://localhost:8082`)
- `ACCOUNT_SERVICE_URL` (default `http://localhost:8080`)
- `LEDGER_SERVICE_URL` (default `http://localhost:8083`)
- `K6_ACCESS_TOKEN`
- `K6_AUTH_EMAIL`
- `K6_AUTH_PASSWORD`
