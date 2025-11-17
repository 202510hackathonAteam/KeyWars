
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

# MemoryStore
resource "google_redis_instance" "redis" {
  name           = "redis"
  tier           = "BASIC"
  memory_size_gb = 2
  region         = var.region
  redis_version  = "REDIS_7_0"
  authorized_network = google_compute_network.vpc_network.id
  connect_mode = "PRIVATE_SERVICE_ACCESS"
  reserved_ip_range = google_compute_global_address.memorystore_ip_range.name
}