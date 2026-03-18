# OAuth2 Authorization Code + PKCE (Local Dev)

This repo includes an internal OIDC provider (`identity-service`) that exposes OAuth2 Authorization Code + PKCE endpoints.

## Endpoints (via api-gateway)

- `GET /oauth/authorize`
- `POST /oauth/token`
- `GET /.well-known/openid-configuration`
- `GET /.well-known/jwks.json`

Admin client registry endpoints:

- `POST /v1/auth/admin/oauth/clients`
- `GET /v1/auth/admin/oauth/clients/{client_id}`
- `GET /v1/auth/admin/oauth/clients`
- `DELETE /v1/auth/admin/oauth/clients/{client_id}`
- `POST /v1/auth/admin/oauth/clients/{client_id}/secret:rotate`

## Gateway auth/routing notes

- The gateway treats `/.well-known/*` and `/oauth/*` as public by default.
- Admin endpoints under `/v1/auth/admin/*` are NOT public and require a valid JWT.
- Default public auth endpoints:
  - `/v1/auth/register`
  - `/v1/auth/login`
  - `/v1/auth/refresh`
  - `/v1/auth/logout`
  - `/v1/auth/logout_all`

## Running the example flow

A helper script is provided:

```bash
BASE_URL=http://localhost:8080 \
EMAIL=user@example.com \
PASSWORD=password123 \
FULL_NAME="User Example" \
REDIRECT_URI=https://app.example/cb \
ADMIN_ACCESS_TOKEN=... \
./scripts/oauth_pkce_flow.sh
```

### What the script does

- Registers a user (best-effort)
- Logs in to get a *user* access token
- Uses `ADMIN_ACCESS_TOKEN` to create a public OAuth client with `REDIRECT_URI`
- Generates PKCE verifier + S256 challenge
- Calls `/oauth/authorize` (currently API-first and requires the user's bearer token)
- Exchanges the code at `/oauth/token`

## Troubleshooting

- If `/oauth/token` is rate limited, wait a second and retry.
- If client creation fails, ensure `ADMIN_ACCESS_TOKEN` includes `auth:admin` permission.
- If authorize fails with redirect URI errors, confirm the client was created with the exact same `REDIRECT_URI`.
