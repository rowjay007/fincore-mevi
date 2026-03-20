# AGENTS.md

This repository is designed to be safe to work on with multiple humans and automated agents. This document defines the rules of engagement.

## What this repo is

- Go monorepo for FinCore services.
- Key services:
  - `services/identity-service`: OIDC discovery + JWKS + browser OAuth authorize UX
  - `services/auth-service`: core auth logic (JWT issuance, OAuth2 PKCE, client registry)
  - `services/api-gateway`: public/private routing + rate limiting
  - `services/account-service`, `services/ledger-service`: CQRS + Event Sourcing domain services

## Non-negotiable invariants

- **JWKS canonical endpoint** is `/.well-known/jwks.json`.
- **OAuth2 endpoints** are:
  - `/oauth/authorize`
  - `/oauth/token`
- **Admin endpoints** must remain private at the gateway.
- **SPIFFE mTLS** boundaries must not be weakened.
  - When enabled, do not bypass mTLS for internal gRPC.
- **Redirect URI security**:
  - Redirect URIs containing fragments (`#...`) must be rejected.

## Development workflow

### Local commands (must stay green)

Run these before pushing:

```bash
# formatting check
 test -z "$(gofmt -l .)"

# static analysis
 go vet ./...
 go install honnef.co/go/tools/cmd/staticcheck@latest
 staticcheck ./...

# tests
 go test ./...
 go test -race ./...

# coverage gate (mirrors CI)
 MIN_COVERAGE=70
 go test ./pkg/security/... ./services/auth-service/infrastructure/grpc/... -coverprofile=coverage.out
 pct=$(go tool cover -func=coverage.out | awk '/total:/ {gsub(/%/,"",$3); print $3}')
 echo "coverage=${pct}% (min ${MIN_COVERAGE}%)"
 awk -v p="$pct" -v min="$MIN_COVERAGE" 'BEGIN { exit (p+0 < min+0) }'
```

### CI

Primary CI workflow is in `.github/workflows/ci.yml`.

## How to make changes

- Prefer small, reviewable commits.
- Add tests for:
  - request validation
  - security constraints
  - error mapping and edge cases
- Update docs when changing:
  - public endpoints
  - environment variables
  - security behavior

## Security guidance

- Never log secrets (passwords, client secrets, access tokens, refresh tokens).
- Avoid reflecting internal errors to OAuth clients unless sanitized.
- Prefer local JWT verification using JWKS.
- Keep rate limits strict on auth endpoints (`/oauth/token`, `/v1/auth/login`, `/v1/auth/register`).

## Agent behavior guidelines

When you are an automated agent working in this repo:

- Do not change public APIs/routes without updating docs and tests.
- Do not add new dependencies without a strong reason.
- If staticcheck/vet fails, fix the root cause (not the symptom).
- If you cannot find a roadmap item, update `roadmap.md` first.

## Commit and summary requirement

For each meaningful change set, include:

- A commit message suggestion.
- An enterprise production summary:
  - why
  - what changed
  - tradeoffs
