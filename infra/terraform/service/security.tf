# SSH用ファイアーウォール
resource "google_compute_firewall" "ssh" {
  name = "ssh-fw"
  network = google_compute_network.vpc_network.name

  direction = "INGRESS" # 内向き
  allow {
    protocol = "tcp"
    ports = ["22"] # SSHのポートを許可
  }  

  target_tags = ["bastion-tag"] # 対象のタグ
  source_ranges = ["35.235.240.0/20"] # IAPが使用する範囲のみ許可
}

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
