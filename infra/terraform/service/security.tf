### CloudSQLのSecret作成
# MySQLユーザーパスワードのsecret作成
resource "google_secret_manager_secret" "mysql_user_password" {
  secret_id = "mysql-user-password"
  replication {
    auto {} # 自動で別リージョンに複製される
  }
  depends_on = [google_project_service.secretmanager_api]
}

# MySQLユーザーパスワードのsecretの値を設定
resource "google_secret_manager_secret_version" "mysql_user_password_version" {
  secret      = google_secret_manager_secret.mysql_user_password.id
  secret_data = var.mysql_user_password
}

# Secret参照権限を追加
resource "google_secret_manager_secret_iam_member" "secretaccess_mysql_user_password" {
  secret_id = google_secret_manager_secret.mysql_user_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
}

# MySQLrootパスワードのsecret作成
resource "google_secret_manager_secret" "mysql_root_password" {
  secret_id = "mysql-root-password"
  replication {
    auto {} # 自動で別リージョンに複製される
  }
  depends_on = [google_project_service.secretmanager_api]
}

# MySQLrootパスワードのsecretの値を設定
resource "google_secret_manager_secret_version" "mysql_root_password_version" {
  secret      = google_secret_manager_secret.mysql_root_password.id
  secret_data = var.mysql_root_password
}

# Secret参照権限を追加
resource "google_secret_manager_secret_iam_member" "secretaccess_mysql_root_password" {
  secret_id = google_secret_manager_secret.mysql_root_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
}


# Redisパスワードのsecret作成
resource "google_secret_manager_secret" "redis_password" {
  secret_id = "redis-password"
  replication {
    auto {} # 自動で別リージョンに複製される
  }
  depends_on = [google_project_service.secretmanager_api]
}

# Redisパスワードのsecretの値を設定
resource "google_secret_manager_secret_version" "redis_password_version" {
  secret      = google_secret_manager_secret.redis_password.id
  secret_data = var.redis_password # 
}

# Secret参照権限を追加
resource "google_secret_manager_secret_iam_member" "secretaccess_redis_password" {
  secret_id = google_secret_manager_secret.redis_password.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.project_number}-compute@developer.gserviceaccount.com"
}

# パスワード更新
resource "null_resource" "update_password" {
  # CloudRunの作成をトリガーとする
  depends_on = [
    module.cloud_run_api, 
    module.cloud_run_websocket, 
    module.cloud_run_migration, 
    module.cloud_run_seed 
  ]

  provisioner "local-exec" {
    command = "bash ${path.root}/../../update_passwords.sh"

    environment = {
      PROJECT_ID      = var.project_id
      REGION          = var.region
      SQL_INSTANCE    = google_sql_database_instance.mysql.name
      SQL_USER        = google_sql_user.mysql_user.name
      CLOUD_RUN_SERVICE_API = module.cloud_run_api.name
      CLOUD_RUN_SERVICE_WS = module.cloud_run_websocket.name
      CLOUD_RUN_JOB_MIGRATION = module.cloud_run_migration.name
      CLOUD_RUN_JOB_SEED = module.cloud_run_seed.name
      MYSQL_USER_PASSWORD_SECRET = google_secret_manager_secret.mysql_user_password.secret_id
      MYSQL_ROOT_PASSWORD_SECRET = google_secret_manager_secret.mysql_root_password.secret_id
    }
  }
}