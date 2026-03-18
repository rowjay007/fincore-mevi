# Vault Agent Auto-Auth (no static VAULT_TOKEN)

## Reason (why)

Using `VAULT_TOKEN` directly in service environment variables re-introduces a long-lived secret distribution problem.

Vault Agent Auto-Auth is the recommended stepping stone to production:

- Workload authenticates using a platform-native method (Kubernetes auth, AppRole, etc.)
- Vault Agent writes a short-lived token to a **token sink file**
- Your service reads the token from `VAULT_TOKEN_FILE`

## Usage (how)

1. Run Vault Agent as a sidecar (Kubernetes recommended).
2. Configure Auto-Auth + token sink file.
3. Set your service env:

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN_FILE=/vault/agent/token
```

Example config template:

- `infra/vault/agent-autoauth-k8s.hcl`

## Notes

- This repo currently supports `VAULT_TOKEN` **or** `VAULT_TOKEN_FILE`.
- SPIFFE-based Vault auth can be added later; Agent Auto-Auth keeps the runtime integration stable while auth method evolves.
