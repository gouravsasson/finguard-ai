terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" { type = string }
variable "region" {
  type    = string
  default = "us-central1"
}
variable "environment" {
  type    = string
  default = "staging"
}

resource "google_artifact_registry_repository" "app" {
  location      = var.region
  repository_id = "finguard-${var.environment}"
  format        = "DOCKER"
}

resource "google_secret_manager_secret" "database_url" {
  secret_id = "finguard-${var.environment}-database-url"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret" "gemini_api_key" {
  secret_id = "finguard-${var.environment}-gemini-api-key"
  replication {
    auto {}
  }
}

resource "google_service_account" "run" {
  account_id   = "finguard-${var.environment}"
  display_name = "FinGuard ${var.environment} runtime"
}

resource "google_sql_database_instance" "postgres" {
  name             = "finguard-${var.environment}"
  database_version = "POSTGRES_16"
  region           = var.region
  settings {
    tier = "db-custom-1-3840"
    backup_configuration {
      enabled = true
    }
    ip_configuration {
      ipv4_enabled = false
    }
  }
  deletion_protection = true
}

resource "google_sql_database" "app" {
  name     = "finguard"
  instance = google_sql_database_instance.postgres.name
}

resource "google_cloud_run_v2_service" "api" {
  name     = "finguard-${var.environment}"
  location = var.region
  template {
    service_account = google_service_account.run.email
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.app.repository_id}/finguard:latest"
      ports {
        container_port = 8080
      }
      env {
        name  = "ENV"
        value = "production"
      }
      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "GEMINI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.gemini_api_key.secret_id
            version = "latest"
          }
        }
      }
    }
  }
}

output "service_name" {
  value = google_cloud_run_v2_service.api.name
}
