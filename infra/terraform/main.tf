terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "6.8.0"
    }
  }
}

provider "google" {
  project = "var.project_id"
  region  = "asia-northeast1"
}

# API有効化
resource "google_project_service" "enable_compute_api" {
  project = var.project_id
  service = "compute.googleapis.com"
}

resource "google_project_service" "enable_DNS_api" {
  project = var.project_id
  service = "dns.googleapis.com" 
}

# VPC作成
resource "google_compute_network" "vpc_network" {
  name                    = "${var.project_id}-vpc"
  auto_create_subnetworks = false
  mtu                     = 1460
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


# CloudStrageバケット作成
resource "google_storage_bucket" "assets" {
  name = "${var.project_id}-bucket"
  location = var.region
  uniform_bucket_level_access = true
  storage_class = "STANDARD"
  force_destroy = true # 削除時に中身も削除する
  # デフォルトページ設定
  website {
    main_page_suffix = "dist/index.html"
    not_found_page   = "404.html"
  }
}

# バケットへのアクセス権
resource "google_storage_bucket_iam_member" "default" {
  bucket = google_storage_bucket.assets.name
	role   = "roles/storage.objectViewer" # 閲覧者ロール
  member = "allUsers" # プリンシバル(対象範囲)
}


### HTTPS用ロードバランサ
# 固定IPアドレス取得
resource "google_compute_global_address" "lb_ip" {
  name = "lb-ip"
}

# バックエンドバケット
resource "google_compute_backend_bucket" "bucket1" {
  name = "cs-bucket"
  description = "CloudStrage bucket"
  bucket_name = google_storage_bucket.assets.name
  enable_cdn = true
}

# CloudRunはここに追加

# urlマップ(バックエンドルール)
resource "google_compute_url_map" "default" {
  name = "url-map"
  # 指定したドメインに対して、使用するpath_matcherを指定
  host_rule {
    hosts = [ "keywars.jp" ]
    path_matcher = "allpaths"
  }
  # パスに応じて選択するバックエンドを指定
  path_matcher {
    name = "allpaths"
    default_service = google_compute_backend_bucket.bucket1.self_link # どれにも該当しないトラフィックの転送先

    # 特定のパターンに合致する場合の転送先
    # path_rule {
    #   paths = ["/api/*"]
    #   service = 
    # }

    # path_rule {
    #   paths = ["/ws/*"]
    #   service = 
    # }
  }
}

# GoogleマネージドSSL証明書の発行
resource "google_compute_managed_ssl_certificate" "default" {
  provider = google
  name = "ssl-cert"
  managed {
    domains = [ "keywars.jp" ]
  }
}

# HTTPS転送ターゲットプロキシ
resource "google_compute_target_https_proxy" "default" {
  name = "https_proxy"
  url_map = google_compute_url_map.default.id
  ssl_certificates = [ google_compute_managed_ssl_certificate.default.name ]
  depends_on = [ google_compute_managed_ssl_certificate.default ]
}

# フロントエンドルール
resource "google_compute_global_forwarding_rule" "default" {
  name = "forwarding-rule"
  ip_address = google_compute_global_address.lb_ip.address
  port_range = "443"
  target = google_compute_target_https_proxy.default.self_link
  load_balancing_scheme = "EXTERNAL"
}

### HTTPSリダイレクト用ロードバランサ
# urlマップ(バックエンドルール)
resource "google_compute_url_map" "http_redirect" {
  name = "http-redirect-map"
  default_url_redirect {
    https_redirect = true
    strip_query = false
  }
}

# HTTP転送ターゲットプロキシ
resource "google_compute_target_http_proxy" "http_proxy" {
  name = "http_proxy"
  url_map = google_compute_url_map.http_redirect.self_link
}

# フロントエンドルール
resource "google_compute_global_forwarding_rule" "http_rule" {
  name = "http-forwarding-rule"
  port_range = "80"
  target = google_compute_target_http_proxy.http_proxy.self_link
  ip_address = google_compute_global_address.lb_ip.address
  load_balancing_scheme = "EXTERNAL"
}

# DNSゾーン作成
resource "google_dns_managed_zone" "zone" {
  name = "${var.project_id}-zone"
  dns_name = "keywars.jp."
}

# Aレコード作成
resource "google_dns_record_set" "A_record" {
  name = google_dns_managed_zone.zone.dns_name
  managed_zone = google_dns_managed_zone.zone.name
  type = "A"
  ttl = "300"
  rrdatas = [ google_compute_global_address.lb_ip.address ]
}

