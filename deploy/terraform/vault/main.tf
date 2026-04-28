terraform {
  required_providers {
    vault = {
      source  = "hashicorp/vault"
      version = "~> 4.0"
    }
  }
}

provider "vault" {
  # Address and Token are typically provided via VAULT_ADDR and VAULT_TOKEN env vars
}

# 1. Enable KV-V2 Secrets Engine
resource "vault_mount" "kvv2" {
  path        = "secret"
  type        = "kv-v2"
  description = "Primary KV-V2 secrets engine for FinCore services"
}

# 2. Enable Kubernetes Auth Method
resource "vault_auth_backend" "kubernetes" {
  type = "kubernetes"
}

resource "vault_kubernetes_auth_backend_config" "config" {
  backend                = vault_auth_backend.kubernetes.path
  kubernetes_host        = "https://kubernetes.default.svc"
  disable_iss_validation = "true"
}

# 3. Define Policies for Services
resource "vault_policy" "identity_service" {
  name   = "identity-service"
  policy = <<EOT
path "secret/data/identity" {
  capabilities = ["read"]
}
EOT
}

resource "vault_policy" "ledger_service" {
  name   = "ledger-service"
  policy = <<EOT
path "secret/data/ledger" {
  capabilities = ["read"]
}
EOT
}

# 4. Create Kubernetes Roles for Services
resource "vault_kubernetes_auth_backend_role" "identity_service" {
  backend                          = vault_auth_backend.kubernetes.path
  role_name                        = "identity-service"
  bound_service_account_names      = ["identity-service"]
  bound_service_account_namespaces = ["fincore"]
  token_policies                   = ["identity-service"]
  token_ttl                        = 3600
}

resource "vault_kubernetes_auth_backend_role" "ledger_service" {
  backend                          = vault_auth_backend.kubernetes.path
  role_name                        = "ledger-service"
  bound_service_account_names      = ["ledger-service"]
  bound_service_account_namespaces = ["fincore"]
  token_policies                   = ["ledger-service"]
  token_ttl                        = 3600
}
