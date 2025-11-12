terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "6.8.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
#----------------------------
# API有効化
#----------------------------

resource "google_project_service" "cloudresourcemanager_api" {
  project            = var.project_id
  service            = "cloudresourcemanager.googleapis.com"
  disable_on_destroy = false # destroy時に無効化しない
}


resource "google_project_service" "compute_api" {
  project            = var.project_id
  service            = "compute.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "DNS_api" {
  project            = var.project_id
  service            = "dns.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudbuild_api" {
  project            = var.project_id
  service            = "cloudbuild.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "secretmanager_api" {
  project            = var.project_id
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "servicenetworking_api" {
  project            = var.project_id
  service            = "servicenetworking.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "sqladmin_api" {
  project            = var.project_id
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudrun_api" {
  project            = var.project_id
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "artifactregistry_api" {
  project            = var.project_id
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}


#----------------------------
#VPC・サブネット
#----------------------------

# VPC作成
resource "google_compute_network" "vpc_network" {
  name                    = "${var.project_id}-vpc"
  auto_create_subnetworks = false
  mtu                     = 1460
}

# VPCピアリング用のPrivateIPを確保
resource "google_compute_global_address" "private_ip_address" {
  name          = "private-ip-address"
  purpose       = "VPC_PEERING" # VPCピアリング用
  address_type  = "INTERNAL"    # 内部(private)IPアドレス
  prefix_length = 16
  network       = google_compute_network.vpc_network.id
  depends_on    = [google_compute_network.vpc_network]
}

# VPCピアリング用PrivateIPとGoogleサービスのネットワークと接続する
resource "google_service_networking_connection" "default" {
  network                 = google_compute_network.vpc_network.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
  depends_on              = [google_project_service.servicenetworking_api, google_compute_global_address.private_ip_address]
}

# サブネット作成
resource "google_compute_subnetwork" "group1" {
  name          = "subnet-connector"
  ip_cidr_range = "10.0.1.0/28" # コネクタ用は/28
  region        = var.region
  network       = google_compute_network.vpc_network.id
}

resource "google_compute_subnetwork" "group2" {
  name          = "subnet-sql"
  ip_cidr_range = "10.0.2.0/24"
  region        = var.region
  network       = google_compute_network.vpc_network.id
}

resource "google_compute_subnetwork" "group3" {
  name          = "subnet-ms"
  ip_cidr_range = "10.0.3.0/24"
  region        = var.region
  network       = google_compute_network.vpc_network.id
}

resource "google_compute_subnetwork" "group4" {
  name          = "vpc-connector"
  ip_cidr_range = "10.0.16.0/20" # Direct VPC Egressようなので広め
  region        = var.region
  network       = google_compute_network.vpc_network.id
}



#----------------------------
# Strage
#----------------------------

# CloudStrageバケット作成
resource "google_storage_bucket" "static" {
  name                        = "${var.project_id}-bucket"
  location                    = var.region
  uniform_bucket_level_access = true
  storage_class               = "STANDARD"
  force_destroy               = true # 削除時に中身も削除する
  # デフォルトページ設定
  website {
    main_page_suffix = "index.html"
    not_found_page   = "404.html"
  }
}

# バケットへのアクセス権
resource "google_storage_bucket_iam_member" "default" {
  bucket = google_storage_bucket.static.name
  role   = "roles/storage.objectViewer" # 閲覧者ロール
  member = "allUsers"                   # プリンシバル(対象範囲)
}

# バケットにファイルアップロード
resource "google_storage_bucket_object" "html" {
  bucket       = google_storage_bucket.static.id
  for_each     = fileset(var.frontend_static_path, "*.html")
  name         = each.value                                  # GCS内でのファイル名
  source       = "${var.frontend_static_path}/${each.value}" # アップロードするファイルのパス
  content_type = "text/html"
}

resource "google_storage_bucket_object" "css" {
  bucket       = google_storage_bucket.static.id
  for_each     = fileset(var.frontend_static_path, "assets/*.css")
  name         = each.value # GCS内でのファイル名
  source       = "${var.frontend_static_path}/${each.value}"
  content_type = "text/css"
}

resource "google_storage_bucket_object" "js" {
  bucket       = google_storage_bucket.static.id
  for_each     = fileset(var.frontend_static_path, "assets/*.js")
  name         = each.value # GCS内でのファイル名
  source       = "${var.frontend_static_path}/${each.value}"
  content_type = "application/javascript"
}

resource "google_storage_bucket_object" "svg" {
  bucket       = google_storage_bucket.static.id
  for_each     = fileset(var.frontend_static_path, "*.svg")
  name         = each.value # GCS内でのファイル名
  source       = "${var.frontend_static_path}/${each.value}"
  content_type = "image/svg+xml"
}

#----------------------------
# ロードバランサ
#----------------------------

### HTTPS用ロードバランサ
# 固定IPアドレス取得
resource "google_compute_global_address" "lb_ip" {
  name = "lb-ip"
}

# バックエンドバケット
resource "google_compute_backend_bucket" "static_bucket" {
  name        = "static-bucket"
  description = "CloudStrage bucket"
  bucket_name = google_storage_bucket.static.name
  enable_cdn  = true
  depends_on  = [google_storage_bucket.static]
}

# サーバーレスNEG
resource "google_compute_region_network_endpoint_group" "cloudrun_api_neg" {
  name                  = "cloudrun-api-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = "cloudrun-api"
  }
}

# バックエンドサービス
resource "google_compute_backend_service" "api_service" {
  name                  = "api-service"
  load_balancing_scheme = "EXTERNAL_MANAGED"

  backend {
    group = google_compute_region_network_endpoint_group.cloudrun_api_neg.id
  }
  #あとでWebsocket用追加？も一個作る？
  # backend {
  #   group = google_compute_region_network_endpoint_group.cloudrun_api_neg.id
  # }

  depends_on = [
    google_project_service.compute_api,
  ]
}

# urlマップ(バックエンドルール)
resource "google_compute_url_map" "default" {
  name            = "url-map"
  default_service = google_compute_backend_bucket.static_bucket.id
  # 指定したドメインに対して、使用するpath_matcherを指定
  host_rule {
    hosts        = ["keywars.jp"]
    path_matcher = "allpaths"
  }
  # パスに応じて選択するバックエンドを指定
  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_bucket.static_bucket.id # どれにも該当しないトラフィックの転送先

    # 特定のパターンに合致する場合の転送先
    # テスト用
    path_rule {
      paths   = ["/hello"]
      service = google_compute_backend_service.api_service.id
    }

    # API用 
    path_rule {
      paths   = ["/api/*"]
      service = google_compute_backend_service.api_service.id
    }

    # WebSocket用 
    # path_rule {
    #   paths = ["/ws/*"]
    #   service = 
    # }
  }
}

# GoogleマネージドSSL証明書の発行
resource "google_compute_managed_ssl_certificate" "default" {
  provider = google
  name     = "ssl-cert"
  managed {
    domains = ["keywars.jp"]
  }
}

# HTTPS転送ターゲットプロキシ
resource "google_compute_target_https_proxy" "default" {
  name             = "https-proxy"
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_managed_ssl_certificate.default.name]
  depends_on       = [google_compute_managed_ssl_certificate.default]
}

# フロントエンドルール
resource "google_compute_global_forwarding_rule" "default" {
  name                  = "forwarding-rule"
  ip_address            = google_compute_global_address.lb_ip.address
  port_range            = "443"
  target                = google_compute_target_https_proxy.default.self_link
  load_balancing_scheme = "EXTERNAL"
}

### HTTPSリダイレクト用ロードバランサ
# urlマップ(バックエンドルール)
resource "google_compute_url_map" "http_redirect" {
  name = "http-redirect-map"
  default_url_redirect {
    https_redirect = true
    strip_query    = false
  }
}

# HTTP転送ターゲットプロキシ
resource "google_compute_target_http_proxy" "http_proxy" {
  name    = "http-proxy"
  url_map = google_compute_url_map.http_redirect.self_link
}

# フロントエンドルール
resource "google_compute_global_forwarding_rule" "http_rule" {
  name                  = "http-forwarding-rule"
  port_range            = "80"
  target                = google_compute_target_http_proxy.http_proxy.self_link
  ip_address            = google_compute_global_address.lb_ip.address
  load_balancing_scheme = "EXTERNAL"
}

# DNSゾーン作成
# resource "google_dns_managed_zone" "zone" {
#   name = "${var.project_id}-zone"
#   dns_name = "keywars.jp."
# }

# Aレコード作成
resource "google_dns_record_set" "A_record" {
  name         = var.dns_record_name
  managed_zone = var.dns_zone_name
  type         = "A"
  ttl          = "300"
  rrdatas      = [google_compute_global_address.lb_ip.address]
}

#----------------------------
# Database
#----------------------------

# CloudSQL
resource "google_sql_database_instance" "mysql" {
  name             = "mysql"
  region           = var.region
  database_version = "MYSQL_8_0"
  root_password    = var.mysql_root_password
  settings {
    tier = "db-f1-micro"
    ip_configuration {
      ipv4_enabled    = "false"                               # パブリックIPv4アドレスを無効
      private_network = google_compute_network.vpc_network.id # 接続するVPC指定
    }
    password_validation_policy {
      min_length                  = 8
      complexity                  = "COMPLEXITY_DEFAULT" # 複雑さ
      reuse_interval              = 0                    # パスワード再利用までの回数
      disallow_username_substring = true                 # パスワードにユーザー名を許可しない
      enable_password_policy      = true                 # パスワードポリシーのON/OFF
    }
  }
  deletion_protection = false # Terraformでの削除から保護しない
  depends_on = [
    google_project_service.sqladmin_api,
    google_service_networking_connection.default
  ]
}

# MySQLデータベース作成
resource "google_sql_database" "mysql_db" {
  name     = var.mysql_database
  instance = google_sql_database_instance.mysql.name
}

# ユーザー作成
resource "google_sql_user" "mysql_user" {
  name     = var.mysql_user
  password = var.mysql_password
  instance = google_sql_database_instance.mysql.name

}

### CloudSQLのSecret作成
# データベースユーザーのsecret作成 
resource "google_secret_manager_secret" "dbuser" {
  secret_id = "dbuser"
  replication {
    auto {} # 自動で別リージョンに複製される
  }
  depends_on = [google_project_service.secretmanager_api]
}

# データベースユーザーのsecretの値を設定
resource "google_secret_manager_secret_version" "dbuser_version" {
  secret      = google_secret_manager_secret.dbuser.id
  secret_data = var.mysql_user
}

# Secret参照権限を追加
resource "google_secret_manager_secret_iam_member" "secretaccess_compute_dbuser" {
  secret_id = google_secret_manager_secret.dbuser.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
}


# データベースパスワードのsecret作成
resource "google_secret_manager_secret" "dbpassword" {
  secret_id = "dbpassword"
  replication {
    auto {} # 自動で別リージョンに複製される
  }
  depends_on = [google_project_service.secretmanager_api]
}

# データベースパスワードのsecretの値を設定
resource "google_secret_manager_secret_version" "dbpassword_version" {
  secret      = google_secret_manager_secret.dbpassword.id
  secret_data = var.mysql_password
}

# Secret参照権限を追加
resource "google_secret_manager_secret_iam_member" "secretaccess_compute_dbpassword" {
  secret_id = google_secret_manager_secret.dbpassword.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
}


# データベース名のsecret作成
resource "google_secret_manager_secret" "dbname" {
  secret_id = "dbname"
  replication {
    auto {} # 自動で別リージョンに複製される
  }
  depends_on = [google_project_service.secretmanager_api]
}

# データベース名のsecretの値を設定
resource "google_secret_manager_secret_version" "dbname_version" {
  secret      = google_secret_manager_secret.dbname.id
  secret_data = var.mysql_database
}

# Secret参照権限を追加
resource "google_secret_manager_secret_iam_member" "secretaccess_compute_dbname" {
  secret_id = google_secret_manager_secret.dbname.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
}

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
resource "google_cloud_run_service_iam_member" "allow_unauthenticated" {
  location = google_cloud_run_v2_service.api.location
  service  = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker" # 呼び出し許可
  member   = "allUsers"
}


data "google_client_openid_userinfo" "current" {}

output "current_account_email" {
  value       = data.google_client_openid_userinfo.current.email
  description = "Terraform実行に使用されたサービスアカウントまたはユーザーのメールアドレス"
}