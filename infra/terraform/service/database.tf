
#----------------------------
# CloudSQL
#----------------------------

# CloudSQL
resource "google_sql_database_instance" "mysql" {
  name             = "mysql"
  region           = var.region
  database_version = "MYSQL_8_0"
  root_password    = var.mysql_root_password # applyの最後で再設定
  settings {
    tier = "db-f1-micro"
    ip_configuration {
      ipv4_enabled       = "false"                               # パブリックIPv4アドレスを無効
      private_network    = google_compute_network.vpc_network.id # 接続するVPC指定
      allocated_ip_range = google_compute_global_address.cloudsql_ip_range.name
    }
    password_validation_policy {
      min_length                  = 8
      complexity                  = "COMPLEXITY_DEFAULT" # 複雑さ
      reuse_interval              = 0                    # パスワード再利用までの回数
      disallow_username_substring = true                 # パスワードにユーザー名を許可しない
      enable_password_policy      = true                 # パスワードポリシーのON/OFF
    }
  }
  deletion_protection = false # Terraformでの削除からの保護(本番はtrue推奨)
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
  password = var.mysql_user_password # applyの最後で再設定
  instance = google_sql_database_instance.mysql.name
}

#----------------------------
# MemoryStore for Redis
#----------------------------

# MemoryStore
resource "google_redis_instance" "redis" {
  name               = "redis"
  tier               = "BASIC"
  memory_size_gb     = 1
  region             = var.region
  redis_version      = "REDIS_7_0"
  authorized_network = google_compute_network.vpc_network.id
  connect_mode       = "PRIVATE_SERVICE_ACCESS"
  reserved_ip_range  = google_compute_global_address.memorystore_ip_range.name
  depends_on = [
    google_project_service.redis_api,
    google_service_networking_connection.default
  ]
}