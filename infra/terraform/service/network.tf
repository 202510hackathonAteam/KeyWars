
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