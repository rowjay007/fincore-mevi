#!/usr/bin/env bash
# fincore-deploy.sh - Production-grade deployment orchestrator
set -euo pipefail

NAMESPACE=${NAMESPACE:-fincore}
VAULT_ADDR=${VAULT_ADDR:-"http://vault.vault.svc:8200"}

echo "--- Initializing FinCore Production Environment ---"

# 1. Create Namespace
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# 2. Setup Vault Policies (Example for identity-service)
echo "--- Configuring Vault Security Policies ---"
# vault policy write identity-service - <<EOF
# path "secret/data/identity" { capabilities = ["read"] }
# EOF

# 3. Setup SPIRE Entries
echo "--- Registering SPIRE Workload Identities ---"
# spire-server entry create -spiffeID spiffe://fincore.local/ns/fincore/sa/identity-service \
#   -parentID spiffe://fincore.local/spire/agent/k8s_psat/cluster/node \
#   -selector k8s:ns:fincore -selector k8s:sa:identity-service

# 4. Deploy Services using Helm
SERVICES=("identity-service" "auth-service" "api-gateway" "ledger-service" "payment-service")

for SVC in "${SERVICES[@]}"; do
  echo "--- Deploying ${SVC} ---"
  helm upgrade --install "${SVC}" ./deploy/charts/fincore-service \
    --namespace "${NAMESPACE}" \
    --set image.repository="fincore/${SVC}" \
    --set vault.role="${SVC}" \
    --set vault.secretPath="secret/data/${SVC}" \
    --values "deploy/values/${SVC}.yaml"
done

echo "--- FinCore Deployment Complete ---"
