# FinCore Operational Runbook: Secret Rotation

## Overview
This runbook describes the procedure for rotating sensitive secrets (Database DSNs, JWT Signing Keys) in the FinCore production environment using HashiCorp Vault.

## Criticality: HIGH
Secret leakage or expiration can cause platform-wide downtime or security breaches.

## Procedure: Database DSN Rotation

### 1. Update Secret in Vault
Access the Vault UI or use the CLI to update the secret:
```bash
vault kv put secret/identity dsn="postgres://user:new-password@db.fincore.local:5432/identity"
```

### 2. Trigger Rolling Restart
Since services use the Vault Agent Sidecar with "render" mode (or restart on change), trigger a rolling restart of the relevant deployment:
```bash
kubectl rollout restart deployment identity-service -n fincore
```

### 3. Verify Connectivity
Monitor the logs for successful DB connection:
```bash
kubectl logs -f deployment/identity-service -n fincore | grep "connected to db"
```

## Procedure: JWT Key Rotation (Emergency)

### 1. Generate New Key
Generate a new Ed25519 key pair.

### 2. Add New Key to Vault
Update the `identity` secret with the new `kid` and `jwt_ed25519_private_key`.

### 3. Update JWKS
The `identity-service` automatically updates its `/.well-known/jwks.json` on restart.

### 4. Restart Services
Restart the `identity-service` followed by dependent services to clear cached keys.

---
**Last Updated:** 2026-04-28
**Owner:** Security Operations Team
