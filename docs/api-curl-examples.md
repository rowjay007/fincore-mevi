# API curl examples

## Auth service

### Register

```bash
curl -sS -X POST http://localhost:8082/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "password": "correct-horse-battery-staple",
    "fullName": "Alice Example"
  }'
```

### Login

```bash
curl -sS -X POST http://localhost:8082/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "alice@example.com",
    "password": "correct-horse-battery-staple"
  }'
```

Capture tokens:

```bash
ACCESS_TOKEN=$(curl -sS -X POST http://localhost:8082/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"correct-horse-battery-staple"}' | jq -r .accessToken)

REFRESH_TOKEN=$(curl -sS -X POST http://localhost:8082/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"correct-horse-battery-staple"}' | jq -r .refreshToken)
```

### Validate token

```bash
curl -sS -X POST http://localhost:8082/v1/auth/validate \
  -H 'Content-Type: application/json' \
  -d '{"accessToken":"'"$ACCESS_TOKEN"'"}'
```

### Refresh token

```bash
curl -sS -X POST http://localhost:8082/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refreshToken":"'"$REFRESH_TOKEN"'"}'
```

### Logout (revoke one refresh token)

```bash
curl -sS -X POST http://localhost:8082/v1/auth/logout \
  -H 'Content-Type: application/json' \
  -d '{"refreshToken":"'"$REFRESH_TOKEN"'"}'
```

### Logout all sessions

Either pass the access token in the body:

```bash
curl -sS -X POST http://localhost:8082/v1/auth/logout-all \
  -H 'Content-Type: application/json' \
  -d '{"accessToken":"'"$ACCESS_TOKEN"'"}'
```

Or pass it via `Authorization` header (forwarded by grpc-gateway):

```bash
curl -sS -X POST http://localhost:8082/v1/auth/logout-all \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

### JWKS

```bash
curl -sS http://localhost:8082/jwks.json | jq
```

## Account service

### Open account

```bash
curl -sS -X POST http://localhost:8080/v1/accounts \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "customerId": "cust-1",
    "idempotencyKey": "open-1"
  }'
```

### Deposit

```bash
curl -sS -X POST http://localhost:8080/v1/accounts/acc-1/deposit \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "accountId": "acc-1",
    "idempotencyKey": "dep-1",
    "amount": {"currency": "USD", "units": 10, "nanos": 0},
    "narration": "fund"
  }'
```

### Get account

```bash
curl -sS -X GET http://localhost:8080/v1/accounts/acc-1 \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Ledger service

### Post entry

```bash
curl -sS -X POST http://localhost:8083/v1/ledger/entries \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "idempotencyKey": "entry-1",
    "entryType": "ENTRY_TYPE_DEPOSIT",
    "account": {"accountId": "acc-1"},
    "amount": {"currency": "USD", "units": 10, "nanos": 0},
    "narration": "seed"
  }'
```

### Get balance

```bash
curl -sS -X GET http://localhost:8083/v1/ledger/accounts/acc-1/balance \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```
