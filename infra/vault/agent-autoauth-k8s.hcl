pid_file = "/tmp/vault-agent.pid"

auto_auth {
  method "kubernetes" {
    mount_path = "auth/kubernetes"
    config = {
      role = "identity-service"
    }
  }

  sink "file" {
    config = {
      path = "/vault/agent/token"
    }
  }
}

cache {
  use_auto_auth_token = true
}

listener "tcp" {
  address = "127.0.0.1:8201"
  tls_disable = true
}

vault {
  address = "http://vault.vault.svc.cluster.local:8200"
}
