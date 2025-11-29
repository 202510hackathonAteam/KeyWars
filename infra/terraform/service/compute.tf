
#----------------------------
# 環境変数
#----------------------------

# MySQL関連の環境変数
locals {
  env_vars = [
    {
      name  = "INSTANCE_CONNECTION_NAME"
      value = google_sql_database_instance.mysql.connection_name
    },
    {
      name  = "MYSQL_USER"
      value = var.mysql_user
    },
    {
      name  = "MYSQL_DATABASE"
      value = var.mysql_database
    },
    {
      name  = "MYSQL_HOST"
      value = google_sql_database_instance.mysql.private_ip_address
    },
    {
      name  = "MYSQL_PORT"
      value = var.mysql_port
    },
    {
      name  = "COOKIE_DOMAIN"
      value = "keywars.jp"
    },
    {
      name  = "COOKIE_SECURE"
      value = true
    },
  ]
}

# Redis関連の環境変数
locals {
  redis_env_vars = [
    {
      name  = "REDIS_ADDR"
      value = "${google_redis_instance.redis.host}:6379"
    },
    {
      name  = "REDIS_DB"
      value = "0"
    }
  ]
}

# SecretManagerから取得する環境変数
locals {
  secret_env_vars = [
    {
      name      = "MYSQL_PASSWORD"
      secret_id = google_secret_manager_secret.mysql_user_password.secret_id
      version   = "latest"
    },
    {
      name      = "MYSQL_ROOT_PASSWORD"
      secret_id = google_secret_manager_secret.mysql_root_password.secret_id
      version   = "latest"
    },
    {
      name      = "REDIS_PASSWORD"
      secret_id = google_secret_manager_secret.redis_password.secret_id
      version   = "latest"

    },
  ]
}

#----------------------------
# CloudRunService
#----------------------------

# API用CloudRun
module "cloud_run_api" {
  source = "./modules/cloudrun_service"

  project_id   = var.project_id
  region       = var.region
  service_name = "cloudrun-api"

  image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/api-image:latest"

  env_vars        = local.env_vars
  redis_env_vars  = local.redis_env_vars
  secret_env_vars = local.secret_env_vars

  volume_mounts = [{
    name       = "cloudsql"
    mount_path = "/cloudsql"
  }]

  vpc_network_name    = google_compute_network.vpc_network.name
  vpc_subnetwork_name = google_compute_subnetwork.vpc_connector.name
  network_tags        = ["api"]

  cloudsql_volume_name     = "cloudsql"
  cloudsql_connection_name = [google_sql_database_instance.mysql.connection_name]

  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# WebSocket用CloudRun
module "cloud_run_websocket" {
  source = "./modules/cloudrun_service"

  project_id   = var.project_id
  region       = var.region
  service_name = "cloudrun-websocket"

  image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/ws-image:latest"

  env_vars        = local.env_vars
  redis_env_vars  = local.redis_env_vars
  secret_env_vars = local.secret_env_vars

  volume_mounts = [{
    name       = "cloudsql"
    mount_path = "/cloudsql"
  }]

  vpc_network_name    = google_compute_network.vpc_network.name
  vpc_subnetwork_name = google_compute_subnetwork.vpc_connector.name
  network_tags        = ["ws"]

  cloudsql_volume_name     = "cloudsql"
  cloudsql_connection_name = [google_sql_database_instance.mysql.connection_name]

  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

#----------------------------
# CloudRunJob
#----------------------------

# マイグレーション用CloudRunJob
module "cloud_run_migration" {
  source = "./modules/cloudrun_job"

  project_id   = var.project_id
  region       = var.region
  service_name = "cloudrun-migration"

  image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/migration-image:latest"

  env_vars        = local.env_vars
  redis_env_vars  = local.redis_env_vars
  secret_env_vars = local.secret_env_vars

  volume_mounts = [{
    name       = "cloudsql"
    mount_path = "/cloudsql"
  }]

  vpc_network_name    = google_compute_network.vpc_network.name
  vpc_subnetwork_name = google_compute_subnetwork.vpc_connector.name
  network_tags        = ["migration"]

  cloudsql_volume_name     = "cloudsql"
  cloudsql_connection_name = [google_sql_database_instance.mysql.connection_name]

  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# 初期データ挿入用CloudRunJob
module "cloud_run_seed" {
  source = "./modules/cloudrun_job"

  project_id   = var.project_id
  region       = var.region
  service_name = "cloudrun-seed"

  image = "${var.region}-docker.pkg.dev/${var.project_id}/${var.ar-repository_name}/seed-image:latest"

  env_vars        = local.env_vars
  redis_env_vars  = local.redis_env_vars
  secret_env_vars = local.secret_env_vars

  volume_mounts = [{
    name       = "cloudsql"
    mount_path = "/cloudsql"
  }]

  vpc_network_name    = google_compute_network.vpc_network.name
  vpc_subnetwork_name = google_compute_subnetwork.vpc_connector.name
  network_tags        = ["seed"]

  cloudsql_volume_name     = "cloudsql"
  cloudsql_connection_name = [google_sql_database_instance.mysql.connection_name]

  depends_on = [
    google_project_service.secretmanager_api,
    google_project_service.cloudrun_api,
    google_project_service.sqladmin_api
  ]
}

# 認証なしでアクセスを許可する(公開する)
resource "google_cloud_run_service_iam_member" "allow_unauthenticated_api" {
  location   = module.cloud_run_api.region
  service    = module.cloud_run_api.name
  role       = "roles/run.invoker" # 呼び出し許可
  member     = "allUsers"
  depends_on = [module.cloud_run_api]
}

resource "google_cloud_run_service_iam_member" "allow_unauthenticated_websocket" {
  location   = module.cloud_run_websocket.region
  service    = module.cloud_run_websocket.name
  role       = "roles/run.invoker" # 呼び出し許可
  member     = "allUsers"
  depends_on = [module.cloud_run_migration]
}

#----------------------------
# Computer Engine
#----------------------------

# 踏み台GCE
resource "google_compute_instance" "bastion" {
  name         = "bastion-vm"
  machine_type = "e2-micro"
  zone         = "us-central1-a" # 無料枠利用のためアイオワ
  tags         = ["bastion-tag"]

  boot_disk {
    initialize_params {
      image = "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-2510-questing-amd64-v20251113"
      size  = 10
    }
  }

  network_interface {
    network    = google_compute_network.vpc_network.name
    subnetwork = google_compute_subnetwork.bastion.name
  }
}