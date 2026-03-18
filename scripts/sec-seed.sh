#!/usr/bin/env bash
set -euo pipefail

SPIRE_COMPOSE_FILE=${SPIRE_COMPOSE_FILE:-docker-compose.spire.yaml}
VAULT_COMPOSE_FILE=${VAULT_COMPOSE_FILE:-docker-compose.vault.yaml}

TRUST_DOMAIN=${SPIFFE_TRUST_DOMAIN:-fincore.local}

DEV_DIR=${DEV_DIR:-.dev}
DEV_SPIRE_AGENT_DIR=${DEV_SPIRE_AGENT_DIR:-${DEV_DIR}/spire/agent}

ACCOUNT_SVID=${ACCOUNT_SVID:-spiffe://${TRUST_DOMAIN}/ns/default/sa/account-service}
LEDGER_SVID=${LEDGER_SVID:-spiffe://${TRUST_DOMAIN}/ns/default/sa/ledger-service}
IDENTITY_SVID=${IDENTITY_SVID:-spiffe://${TRUST_DOMAIN}/ns/default/sa/identity-service}
AGENT_SVID=${AGENT_SVID:-spiffe://${TRUST_DOMAIN}/spire/agent}

ACCOUNT_SELECTOR=${ACCOUNT_SELECTOR:-docker:label:com.fincore.service:account-service}
LEDGER_SELECTOR=${LEDGER_SELECTOR:-docker:label:com.fincore.service:ledger-service}
IDENTITY_SELECTOR=${IDENTITY_SELECTOR:-docker:label:com.fincore.service:identity-service}

VAULT_TOKEN=${VAULT_TOKEN:-root}
VAULT_KV_MOUNT=${VAULT_KV_MOUNT:-secret}
VAULT_IDENTITY_JWT_SECRET_PATH=${VAULT_IDENTITY_JWT_SECRET_PATH:-identity}
VAULT_ACCOUNT_DB_DSN_SECRET_PATH=${VAULT_ACCOUNT_DB_DSN_SECRET_PATH:-account}
VAULT_LEDGER_DB_DSN_SECRET_PATH=${VAULT_LEDGER_DB_DSN_SECRET_PATH:-ledger}

ACCOUNT_DB_DSN=${ACCOUNT_DB_DSN:-postgres://user:password@localhost:5432/fincore?sslmode=disable}
LEDGER_DB_DSN=${LEDGER_DB_DSN:-postgres://user:password@localhost:5432/fincore?sslmode=disable}

ensure_running() {
  docker compose -f "${SPIRE_COMPOSE_FILE}" ps >/dev/null
  docker compose -f "${VAULT_COMPOSE_FILE}" ps >/dev/null
}

spire_exec() {
  docker compose -f "${SPIRE_COMPOSE_FILE}" exec -T spire-server /opt/spire/bin/spire-server "$@"
}

vault_exec() {
  docker compose -f "${VAULT_COMPOSE_FILE}" exec -T -e VAULT_TOKEN="${VAULT_TOKEN}" vault vault "$@"
}

ensure_spire_join_token() {
  local token
  token=$(spire_exec token generate -spiffeID "${AGENT_SVID}" -ttl 24h -format json | sed -n 's/.*"value"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)
  if [[ -z "${token}" ]]; then
    echo "failed to generate spire join token" >&2
    exit 1
  fi

  mkdir -p "${DEV_SPIRE_AGENT_DIR}"
  cat >"${DEV_SPIRE_AGENT_DIR}/agent.conf" <<EOF
agent {
  data_dir = "/run/spire/data"
  log_level = "INFO"
  server_address = "spire-server"
  server_port = "8080"
  socket_path = "/run/spire/sockets/agent.sock"
  trust_domain = "${TRUST_DOMAIN}"
}

plugins {
  NodeAttestor "join_token" {
    plugin_data {
      join_token = "${token}"
    }
  }

  WorkloadAttestor "docker" {
    plugin_data {
      docker_socket_path = "/var/run/docker.sock"
    }
  }

  KeyManager "memory" {
    plugin_data {}
  }
}
EOF

  docker compose -f "${SPIRE_COMPOSE_FILE}" restart spire-agent >/dev/null
}

spire_entry_exists() {
  local spiffe_id=$1
  spire_exec entry show -spiffeID "${spiffe_id}" >/dev/null 2>&1
}

create_spire_entry_if_missing() {
  local spiffe_id=$1
  local selector=$2

  if spire_entry_exists "${spiffe_id}"; then
    return 0
  fi

  spire_exec entry create \
    -spiffeID "${spiffe_id}" \
    -parentID "${AGENT_SVID}" \
    -selector "${selector}" >/dev/null
}

generate_identity_ed25519_priv_b64url() {
  cat <<'EOF' | go run -
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Print(base64.RawURLEncoding.EncodeToString(priv))
}
EOF
}

seed_vault_kv() {
  if ! vault_exec secrets list -format=json | grep -q '"'"${VAULT_KV_MOUNT}"'/"'; then
    vault_exec secrets enable -path="${VAULT_KV_MOUNT}" kv-v2 >/dev/null
  fi

  local kid
  kid=${IDENTITY_JWT_KID:-dev}
  local priv
  priv=$(generate_identity_ed25519_priv_b64url)

  vault_exec kv put "${VAULT_KV_MOUNT}/${VAULT_IDENTITY_JWT_SECRET_PATH}" \
    kid="${kid}" \
    jwt_ed25519_private_key="${priv}" >/dev/null

  vault_exec kv put "${VAULT_KV_MOUNT}/${VAULT_ACCOUNT_DB_DSN_SECRET_PATH}" dsn="${ACCOUNT_DB_DSN}" >/dev/null
  vault_exec kv put "${VAULT_KV_MOUNT}/${VAULT_LEDGER_DB_DSN_SECRET_PATH}" dsn="${LEDGER_DB_DSN}" >/dev/null
}

main() {
  ensure_running

  ensure_spire_join_token

  create_spire_entry_if_missing "${ACCOUNT_SVID}" "${ACCOUNT_SELECTOR}"
  create_spire_entry_if_missing "${LEDGER_SVID}" "${LEDGER_SELECTOR}"
  create_spire_entry_if_missing "${IDENTITY_SVID}" "${IDENTITY_SELECTOR}"

  seed_vault_kv

  echo "Security bootstrap seeded:"
  echo "- SPIRE entries: account-service, ledger-service, identity-service"
  echo "- Vault KV v2: ${VAULT_KV_MOUNT}/${VAULT_IDENTITY_JWT_SECRET_PATH}, ${VAULT_KV_MOUNT}/${VAULT_ACCOUNT_DB_DSN_SECRET_PATH}, ${VAULT_KV_MOUNT}/${VAULT_LEDGER_DB_DSN_SECRET_PATH}"
}

main "$@"
