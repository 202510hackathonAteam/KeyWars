
# #----------------------------
# # CloudStrage
# #----------------------------

# # CloudStrageバケット作成
# resource "google_storage_bucket" "static" {
#   name                        = "${var.project_id}-bucket"
#   location                    = var.region
#   uniform_bucket_level_access = true
#   storage_class               = "STANDARD"
#   force_destroy               = true # 削除時に中身も削除する
#   # デフォルトページ設定
#   website {
#     main_page_suffix = "index.html"
#     not_found_page   = "index.html"
#   }
# }

# # バケットへのアクセス権
# resource "google_storage_bucket_iam_member" "default" {
#   bucket = google_storage_bucket.static.name
#   role   = "roles/storage.objectViewer" # 閲覧者ロール
#   member = "allUsers"                   # プリンシバル(対象範囲)
# }

# # 拡張子別にバケットにファイルアップロード
# resource "google_storage_bucket_object" "html" {
#   bucket       = google_storage_bucket.static.id
#   for_each     = fileset(var.frontend_static_path, "*.html")
#   name         = each.value                                  # GCS内でのファイル名
#   source       = "${var.frontend_static_path}/${each.value}" # アップロードするファイルのパス
#   content_type = "text/html"
# }

# resource "google_storage_bucket_object" "css" {
#   bucket       = google_storage_bucket.static.id
#   for_each     = fileset(var.frontend_static_path, "assets/*.css")
#   name         = each.value # GCS内でのファイル名
#   source       = "${var.frontend_static_path}/${each.value}"
#   content_type = "text/css"
# }

# resource "google_storage_bucket_object" "js" {
#   bucket       = google_storage_bucket.static.id
#   for_each     = fileset(var.frontend_static_path, "assets/*.js")
#   name         = each.value # GCS内でのファイル名
#   source       = "${var.frontend_static_path}/${each.value}"
#   content_type = "application/javascript"
# }

# resource "google_storage_bucket_object" "svg" {
#   bucket       = google_storage_bucket.static.id
#   for_each     = fileset(var.frontend_static_path, "*.svg")
#   name         = each.value # GCS内でのファイル名
#   source       = "${var.frontend_static_path}/${each.value}"
#   content_type = "image/svg+xml"
# }

# resource "google_storage_bucket_object" "jpeg" {
#   bucket       = google_storage_bucket.static.id
#   for_each     = fileset(var.frontend_static_path, "img/*.jpeg")
#   name         = "public/${each.value}" # GCS内でのファイル名
#   source       = "${var.frontend_static_path}/${each.value}"
#   content_type = "image/jpeg"
# }