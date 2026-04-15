variable "cluster_name" {
  type        = string
  description = "Name of the CockroachDB cluster"
}

variable "regions" {
  type        = list(string)
  description = "List of regions for the multi-region cluster"
  default     = ["us-east-1", "eu-west-1", "ap-southeast-1"]
}

resource "cockroach_cluster" "main" {
  name           = var.cluster_name
  cloud_provider = "AWS"
  
  {{- /* Mastery: Multi-region configuration for Geo-partitioning support */}}
  {{- range .regions }}
  region {
    name = .
    node_count = 3
  }
  {{- end }}
}

output "cluster_endpoint" {
  value = cockroach_cluster.main.endpoint
}
