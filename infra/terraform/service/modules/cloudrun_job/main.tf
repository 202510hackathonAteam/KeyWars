resource "google_cloud_run_v2_job" "default" {
  project  = var.project_id
  name     = var.service_name
  location = var.region

  deletion_protection = var.deletion_protection # 削除保護(本番ではtrue推奨)

  template {
    template {
      containers {
        image = var.image

        # MySQL用環境変数を展開
        dynamic "env" {
          for_each = var.mysql_env_vars
          content {
            name  = env.value.name
            value = env.value.value
          }
        }
        # Redis用環境変数を展開
        dynamic "env" {
          for_each = var.redis_env_vars
          content {
            name  = env.value.name
            value = env.value.value
          }
        }
        # SecretManagerからの環境変数を展開
        dynamic "env" {
          for_each = var.secret_env_vars
          content {
            name = env.value.name
            value_source {
              secret_key_ref {
                secret  = env.value.secret_id
                version = env.value.version
              }
            }
          }
        }

        # ボリュームマウント設定
        dynamic "volume_mounts" {
          for_each = var.volume_mounts
          content {
            name  = volume_mounts.value.name
            mount_path = volume_mounts.value.mount_path
          }
        }
      }
      vpc_access {
        # Direct VPC Egress使用
        network_interfaces {
          network    = var.vpc_network_name
          subnetwork = var.vpc_subnetwork_name
          tags       = var.network_tags
        }
      }

      # ボリューム定義
      volumes {
        name = var.cloudsql_volume_name
        cloud_sql_instance {
          instances = var.cloudsql_connection_name
        }
      }
    }
  }

  client  = "terraform"
  depends_on = [var.depends_on_services]

  lifecycle {
    # 変更を無視する
    ignore_changes = [
      client,
    client_version,
      template[0].template[0].containers[0].env, # 環境変数の変更を無視する
      template[0].template[0].containers[0].image, # イメージの変更
     ]
  }
}
