
#----------------------------
# CloudRun
#----------------------------

# MySQL関連の環境変数を定義
locals {
  mysql_env_vars = [
    {
      name  = "INSTANCE_CONNECTION_NAME"
      value = google_sql_database_instance.mysql.connection_name
    },
    {
      name = "MYSQL_USER"
      value_source = {
        secret_key_ref = {
          secret  = google_secret_manager_secret.dbuser.secret_id
          version = google_secret_manager_secret_version.dbuser_version.version
        }
      }
    },
    {
      name = "MYSQL_PASSWORD"
      value_source = {
        secret_key_ref = {
          secret  = google_secret_manager_secret.dbpassword.secret_id
          version = google_secret_manager_secret_version.dbpassword_version.version
        }
      }
    },
    {
      name = "MYSQL_DATABASE"
      value_source = {
        secret_key_ref = {
          secret  = google_secret_manager_secret.dbname.secret_id
          version = google_secret_manager_secret_version.dbname_version.version
        }
      }
    },
    {
      name  = "MYSQL_HOST"
      value = google_sql_database_instance.mysql.private_ip_address
    },
    {
      name  = "MYSQL_PORT"
      value = "3306"
    },
    {
      name  = "REDIS_ADDR"
      value = "${google_redis_instance.redis.host}:6379"
    },
    {
      name  = "REDIS_PASSWORD"
      value = "redispass"
    },
    {
      name  = "REDIS_DB"
      value = "0"
    }

  ]
}

# API用CloudRun
resource "google_cloud_run_v2_service" "api" {
  project  = var.project_id
  name     = "cloudrun-api"
  location = var.region

  deletion_protection = false # 削除保護(本番ではtrue推奨)

  template {
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/api-image:latest"

      # MySQL用環境変数を展開
      dynamic "env" {
        for_each = local.mysql_env_vars
        content {
          name = env.value.name
          # 値がvalueの場合
          value = try(env.value.value, null)
          # 値がvalue_sourceの場合
          dynamic "value_source" {
            for_each = try([env.value.value_source], [])
            content {
              secret_key_ref {
                secret  = value_source.value.secret_key_ref.secret
                version = value_source.value.secret_key_ref.version

              }
            }

          }
        }
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
    }
    vpc_access {
      # Direct VPC Egress使用
      network_interfaces {
        network    = google_compute_network.vpc_network.name
        subnetwork = google_compute_subnetwork.group4.name
        tags       = ["api"]
      }
    }


    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.mysql.connection_name]
      }
    }
  }

  ingress = "INGRESS_TRAFFIC_ALL" # IAMチェックを無効化
  client  = "terraform"
  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# WebSocket用CloudRun
resource "google_cloud_run_v2_service" "websocket" {
  project  = var.project_id
  name     = "cloudrun-websocket"
  location = var.region

  deletion_protection = false # 削除保護(本番ではtrue推奨)

  template {
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/api-image:latest"

      # MySQL用環境変数を展開
      dynamic "env" {
        for_each = local.mysql_env_vars
        content {
          name = env.value.name
          # 値がvalueの場合
          value = try(env.value.value, null)
          # 値がvalue_sourceの場合
          dynamic "value_source" {
            for_each = try([env.value.value_source], [])
            content {
              secret_key_ref {
                secret  = value_source.value.secret_key_ref.secret
                version = value_source.value.secret_key_ref.version

              }
            }

          }
        }
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
    }
    vpc_access {
      # Direct VPC Egress使用
      network_interfaces {
        network    = google_compute_network.vpc_network.name
        subnetwork = google_compute_subnetwork.group4.name
        tags       = ["websocket"]
      }
    }


    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.mysql.connection_name]
      }
    }
  }
  ingress = "INGRESS_TRAFFIC_ALL" # IAMチェックを無効化
  client  = "terraform"
  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# マイグレーション用CloudRunJob
resource "google_cloud_run_v2_job" "migration" {
  project  = var.project_id
  name     = "cloudrun-migration"
  location = var.region

  deletion_protection = false

  template {
    template {
      containers {
        image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/migration-image:latest"
        dynamic "env" {
          for_each = local.mysql_env_vars
          content {
            name = env.value.name
            # 値がvalueの場合
            value = try(env.value.value, null)
            # 値がvalue_sourceの場合
            dynamic "value_source" {
              for_each = try([env.value.value_source], [])
              content {
                secret_key_ref {
                  secret  = value_source.value.secret_key_ref.secret
                  version = value_source.value.secret_key_ref.version
                }
              }
            }
          }
        }
        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
      }

      vpc_access {
        # Direct VPC Egress使用
        network_interfaces {
          network    = google_compute_network.vpc_network.name
          subnetwork = google_compute_subnetwork.group4.name
          tags       = ["migration"]
        }
      }


      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.mysql.connection_name]
        }
      }
    }
  }

  # ingress = "INGRESS_TRAFFIC_ALL" # IAMチェックを無効化
  client = "terraform"
  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# 初期データ挿入用CloudRunJob
resource "google_cloud_run_v2_job" "seed" {
  project  = var.project_id
  name     = "cloudrun-seed"
  location = var.region

  deletion_protection = false

  template {
    template {
      containers {
        image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/seed-image:latest"
        dynamic "env" {
          for_each = local.mysql_env_vars
          content {
            name = env.value.name
            # 値がvalueの場合
            value = try(env.value.value, null)
            # 値がvalue_sourceの場合
            dynamic "value_source" {
              for_each = try([env.value.value_source], [])
              content {
                secret_key_ref {
                  secret  = value_source.value.secret_key_ref.secret
                  version = value_source.value.secret_key_ref.version

                }
              }

            }
          }
        }
        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
      }
      vpc_access {
        # Direct VPC Egress使用
        network_interfaces {
          network    = google_compute_network.vpc_network.name
          subnetwork = google_compute_subnetwork.group4.name
          tags       = ["seed"]
        }
      }

      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.mysql.connection_name]
        }
      }
    }
  }
  client = "terraform"
  depends_on = [

    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# 認証なしでアクセスを許可する(公開する)
resource "google_cloud_run_service_iam_member" "allow_unauthenticated_api" {
  location = google_cloud_run_v2_service.api.location
  service  = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker" # 呼び出し許可
  member   = "allUsers"
}

resource "google_cloud_run_service_iam_member" "allow_unauthenticated_websocket" {
  location = google_cloud_run_v2_service.websocket.location
  service  = google_cloud_run_v2_service.websocket.name
  role     = "roles/run.invoker" # 呼び出し許可
  member   = "allUsers"
}