
#----------------------------
# Firewall
#----------------------------

# SSH用ファイアーウォール
resource "google_compute_firewall" "ssh" {
  name    = "ssh-fw"
  network = google_compute_network.vpc_network.name

  direction = "INGRESS" # 内向き
  allow {
    protocol = "tcp"
    ports    = ["22"] # SSHのポートを許可
  }

  target_tags   = ["bastion-tag"]     # 対象のタグ
  source_ranges = ["35.235.240.0/20"] # IAPが使用する範囲のみ許可
}

#----------------------------
# CloudArmor
#----------------------------

resource "google_compute_security_policy" "default" {
  name = "${var.project_name}-security-policy"
  description = "OWASP Top 10 protection with Cloud Armor"

  # デフォルトルール(すべて許可)
  rule {
    action = "allow"
    priority = 2147483647 # 優先度
    match {
      versioned_expr = "SRC_IPS_V1" # IPアドレスベースのマッチング式
      config {
        src_ip_ranges = ["*"] # すべてのIPアドレスが対象
      }
    }
  }

  # レート制限（DDoS対策）
  rule {
    action   = "rate_based_ban" # 閾値を超えたIPを一定時間BANする
    priority = 1000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      conform_action = "allow" # 制限内のリクエストは許可
      exceed_action  = "deny(429)" # 制限超過時は429エラーを返す
      enforce_on_key = "IP" # IPアドレスごとにカウント
      
      rate_limit_threshold {
        count        = 300 # IPアドレス1つにつき、60秒に300アクセスまで
        interval_sec = 60
      }
      ban_duration_sec = 600 # BANする時間
    }
  }

  # SQLインジェクション、XSS対策
  rule {
    action = "deny(403)"
    priority = 2000
    match {
      expr {
        expression = <<-EOT
                        evaluatePreconfiguredExpr("sqli-v33-stable")
                        || evaluatePreconfiguredExpr("xss-v33-stable")
        EOT
      }
    }
  }

  # 自動DDos検知(トラフィックパターンを学習して異常なトラフィックを自動検出・遮断)
  adaptive_protection_config {
    layer_7_ddos_defense_config {
      enable = true
      rule_visibility = "STANDARD"
    }
  }
}


#----------------------------
# SecretManager
#----------------------------

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
