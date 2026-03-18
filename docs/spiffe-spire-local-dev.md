# SPIFFE/SPIRE local dev scaffold

This repo includes a minimal SPIRE dev stack to bootstrap SPIFFE identities (SVIDs) that we’ll later use for service-to-service mTLS (direct gRPC mTLS, Envoy sidecars, or Istio integration).

## Goals

- Establish a **trust domain** (`spiffe://fincore.local`).
- Run a local **SPIRE server + agent**.
- Demonstrate how we will issue SVIDs for services using selectors.

## What’s included

- `docker-compose.spire.yaml`
- `infra/spire/server/server.conf`
- `infra/spire/agent/agent.conf`

## Run SPIRE locally

1. Start the SPIRE server + agent:

```bash
docker compose -f docker-compose.spire.yaml up
```

2. In another terminal, generate a join token on the SPIRE server (the agent is configured to use `dev-join-token`):

```bash
docker compose -f docker-compose.spire.yaml exec spire-server \
  /opt/spire/bin/spire-server token generate -spiffeID spiffe://fincore.local/spire/agent -ttl 1h -format json
```

Set the agent join token to match.

## Register workload identities (example)

Workload identities are issued based on selectors discovered by the agent. With the Docker workload attestor enabled, you can register entries for containers using Docker label selectors.

Example: issue an SVID for `account-service` containers labeled `com.fincore.service=account-service`:

```bash
docker compose -f docker-compose.spire.yaml exec spire-server \
  /opt/spire/bin/spire-server entry create \
  -spiffeID spiffe://fincore.local/ns/default/sa/account-service \
  -parentID spiffe://fincore.local/spire/agent \
  -selector docker:label:com.fincore.service:account-service
```

Repeat similarly for:

- `spiffe://fincore.local/ns/default/sa/ledger-service`
- `spiffe://fincore.local/ns/default/sa/identity-service`

## Next wiring step (not implemented yet)

Once service containers exist with labels and the agent is running on the same Docker host, we’ll wire the issued SVIDs into mTLS by one of:

- **Direct gRPC mTLS**: use SPIFFE TLS (go-spiffe) to create `grpc.Creds` from the Workload API socket.
- **Envoy sidecars**: use SDS + Workload API socket to provision certs.
- **Istio**: move to Kubernetes and use Istio + SPIRE integration for SPIFFE identities.

## mTLS authorization hardening (allowlists)

By default, the in-process SPIFFE mTLS helper authorizes any peer in the configured trust domain.

You can optionally enable **least-privilege** by setting allowlists:

```bash
export SPIFFE_MTLS_CLIENT_ALLOWED_SVIDS=spiffe://fincore.local/ns/default/sa/ledger-service
export SPIFFE_MTLS_SERVER_ALLOWED_SVIDS=spiffe://fincore.local/ns/default/sa/account-service
```

Notes:

- `SPIFFE_MTLS_CLIENT_ALLOWED_SVIDS` controls which server SVID(s) the client will accept.
- `SPIFFE_MTLS_SERVER_ALLOWED_SVIDS` controls which client SVID(s) the server will accept.
- Values are comma-separated SPIFFE IDs.
