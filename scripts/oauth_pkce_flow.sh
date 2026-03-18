#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
EMAIL="${EMAIL:-user@example.com}"
PASSWORD="${PASSWORD:-password123}"
FULL_NAME="${FULL_NAME:-User Example}"
REDIRECT_URI="${REDIRECT_URI:-https://app.example/cb}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need curl
need jq
need openssl

b64url() {
  openssl base64 -A | tr '+/' '-_' | tr -d '='
}

sha256_b64url() {
  openssl dgst -sha256 -binary | b64url
}

rand_b64url() {
  openssl rand -hex 32 | tr -d '\n'
}

echo "==> Register (idempotent-ish; may fail if already exists)"
set +e
curl -sS -X POST "$BASE_URL/v1/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"full_name\":\"$FULL_NAME\"}" | jq .
set -e

echo "==> Login"
TOKENS=$(curl -sS -X POST "$BASE_URL/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
ACCESS_TOKEN=$(echo "$TOKENS" | jq -r .access_token)
if [[ "$ACCESS_TOKEN" == "null" || -z "$ACCESS_TOKEN" ]]; then
  echo "login failed: $TOKENS" >&2
  exit 1
fi

echo "==> Create OAuth client (admin-only)."
echo "    This script expects you to already have an admin access token."
echo "    Set ADMIN_ACCESS_TOKEN env var (token must include auth:admin)."
ADMIN_ACCESS_TOKEN="${ADMIN_ACCESS_TOKEN:-}"
if [[ -z "$ADMIN_ACCESS_TOKEN" ]]; then
  echo "ADMIN_ACCESS_TOKEN is required to create a client (admin endpoint)." >&2
  exit 1
fi

CLIENT=$(curl -sS -X POST "$BASE_URL/v1/auth/admin/oauth/clients" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $ADMIN_ACCESS_TOKEN" \
  -d "{\"name\":\"local-dev-client\",\"type\":\"public\",\"redirect_uris\":[\"$REDIRECT_URI\"],\"allowed_scopes\":[\"openid\"]}")
CLIENT_ID=$(echo "$CLIENT" | jq -r .client.client_id)
if [[ "$CLIENT_ID" == "null" || -z "$CLIENT_ID" ]]; then
  echo "client create failed: $CLIENT" >&2
  exit 1
fi

echo "==> PKCE: generate verifier + challenge"
CODE_VERIFIER=$(rand_b64url | b64url)
CODE_CHALLENGE=$(printf '%s' "$CODE_VERIFIER" | sha256_b64url)
STATE=$(rand_b64url)

ENC_REDIRECT_URI=$(python - <<PY
import urllib.parse
print(urllib.parse.quote("$REDIRECT_URI", safe=""))
PY
)

echo "==> OAuth authorize (API-first: requires a user Authorization bearer token)"
AUTHZ_URL="$BASE_URL/oauth/authorize?response_type=code&client_id=$CLIENT_ID&redirect_uri=$ENC_REDIRECT_URI&scope=openid&state=$STATE&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256"
RESP_HEADERS=$(mktemp)
HTTP_CODE=$(curl -sS -o /dev/null -D "$RESP_HEADERS" -w '%{http_code}' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$AUTHZ_URL")
if [[ "$HTTP_CODE" != "200" && "$HTTP_CODE" != "302" ]]; then
  echo "authorize failed: HTTP $HTTP_CODE" >&2
  cat "$RESP_HEADERS" >&2
  exit 1
fi

# grpc-gateway likely returns JSON {code,state,redirect_url}
AUTHZ_BODY=$(curl -sS \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$AUTHZ_URL")
CODE=$(echo "$AUTHZ_BODY" | jq -r .code)
if [[ "$CODE" == "null" || -z "$CODE" ]]; then
  echo "authorize did not return code: $AUTHZ_BODY" >&2
  exit 1
fi

echo "==> Exchange code for tokens"
TOKEN=$(curl -sS -X POST "$BASE_URL/oauth/token" \
  -H 'Content-Type: application/json' \
  -d "{\"grant_type\":\"authorization_code\",\"code\":\"$CODE\",\"redirect_uri\":\"$REDIRECT_URI\",\"client_id\":\"$CLIENT_ID\",\"code_verifier\":\"$CODE_VERIFIER\"}")

echo "$TOKEN" | jq .

echo "==> Done"
rm -f "$RESP_HEADERS"
