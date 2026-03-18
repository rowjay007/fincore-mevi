# Vault local dev scaffold

This repo includes a minimal Vault dev setup intended to support Phase 2 secret management (JWT signing keys, DB credentials, API keys) and to prepare for a production posture where services authenticate to Vault using workload identity (SPIFFE).

## Reason (why)

- Avoid storing long-lived secrets in environment variables or repos.
- Centralize secret rotation and audit.
- Prepare for SPIFFE-authenticated workloads (no static Vault tokens on services).

## Usage (local dev)

Start Vault in dev mode:

```bash
docker compose -f docker-compose.vault.yaml up
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

Create an example KV secret (v2):

```bash
vault secrets enable -path=secret kv-v2
vault kv put secret/identity jwt_ed25519_private_key="..." kid="dev"
```

Read it back:

```bash
vault kv get secret/identity
```

## Next step (not implemented yet)

- Enable Vault Kubernetes auth (or Vault Agent) in cluster.
- Enable Vault SPIFFE auth method (or alternative) so services can authenticate using SVIDs.
- Replace `*_JWT_ED25519_PRIVATE_KEY` env vars with a startup fetch from Vault (with caching and rotation strategy).
