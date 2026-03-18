# fincore-mevi

## Identity + JWKS

The platform uses JWTs signed by the internal `identity-service`. Other services verify tokens locally using the identity JWKS endpoint.

Canonical JWKS URL:

```bash
export AUTH_JWKS_URL=http://localhost:8084/.well-known/jwks.json
```
