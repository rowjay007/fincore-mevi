# Vault local dev scaffold

This repo includes a minimal Vault dev setup intended to support Phase 2 secret management (JWT signing keys, DB credentials, API keys) and to prepare for a production posture where services authenticate to Vault using workload identity (SPIFFE).

## Reason (why)

- Avoid storing long-lived secrets in environment variables or repos.
- Centralize secret rotation and audit.
- Prepare for SPIFFE-authenticated workloads (no static Vault tokens on services).

## Usage (local dev)

Start Vault in dev mode:

```bash
make sec-up
```

Vault will be available at:

- `http://localhost:8200`

Default dev root token:

- `root`

Export environment variables for the Vault CLI:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
```

If you are using Vault Agent, you can avoid exporting `VAULT_TOKEN` and instead set `VAULT_TOKEN_FILE`. See `docs/vault-agent-autoauth.md`.

Create an example KV secret (v2):

```bash
vault secrets enable -path=secret kv-v2
vault kv put secret/identity jwt_ed25519_private_key="..." kid="dev"
```

Read it back:

```bash
vault kv get secret/identity
```

If you prefer automation, use:

```bash
make sec-seed
```

## Identity-service wiring (opt-in)

### Reason (why)

This allows running `identity-service` without embedding JWT signing keys in process environment variables, while keeping the default env-var flow intact.

### Usage (how)

`identity-service` will only attempt to read Vault when:

- `VAULT_ADDR` is set
- AND either `VAULT_TOKEN` or `VAULT_TOKEN_FILE` is set
- AND `IDENTITY_JWT_KID` / `IDENTITY_JWT_ED25519_PRIVATE_KEY` are not set

Environment variables:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
export VAULT_TOKEN_FILE=/vault/agent/token
export VAULT_KV_MOUNT=secret
export VAULT_IDENTITY_JWT_SECRET_PATH=identity
```

Expected Vault KV v2 fields at `secret/data/identity`:

- `kid`
- `jwt_ed25519_private_key`

## Account-service wiring (opt-in)

### Reason (why)

This allows running `account-service` without setting `ACCOUNT_DB_DSN` directly in the environment.

### Usage (how)

`account-service` will only attempt to read Vault when:

- `VAULT_ADDR` is set
- AND either `VAULT_TOKEN` or `VAULT_TOKEN_FILE` is set
- AND `ACCOUNT_DB_DSN` is not set

Environment variables:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
export VAULT_TOKEN_FILE=/vault/agent/token
export VAULT_KV_MOUNT=secret
export VAULT_ACCOUNT_DB_DSN_SECRET_PATH=account
```

Expected Vault KV v2 field at `secret/data/account`:

- `dsn`

## Ledger-service wiring (opt-in)

### Reason (why)

This allows running `ledger-service` without setting `LEDGER_DB_DSN` directly in the environment.

### Usage (how)

`ledger-service` will only attempt to read Vault when:

- `VAULT_ADDR` is set
- AND either `VAULT_TOKEN` or `VAULT_TOKEN_FILE` is set
- AND `LEDGER_DB_DSN` is not set

Environment variables:

```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=root
export VAULT_TOKEN_FILE=/vault/agent/token
export VAULT_KV_MOUNT=secret
export VAULT_LEDGER_DB_DSN_SECRET_PATH=ledger
```

Expected Vault KV v2 field at `secret/data/ledger`:

- `dsn`

## Next step (not implemented yet)

- Enable Vault Kubernetes auth (or Vault Agent) in cluster.
- Enable Vault SPIFFE auth method (or alternative) so services can authenticate using SVIDs.
- Replace `*_JWT_ED25519_PRIVATE_KEY` env vars with a startup fetch from Vault (with caching and rotation strategy).
