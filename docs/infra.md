## ディレクトリ構成

infra/
├── terraform/
│   ├── iam/						# Terraform用ユーザーと権限を定義
│   │   ├── main.tf
│   │   ├── outputs.tf
│   │   └── variables.tf
│   │   
│   └── service/					# 構築するリソースを定義
│       ├── api.tf					# 有効化するAPI
│       ├── compute.tf				# CloudRun、ComputerEngine
│       ├── database.tf				# CloudSQL、MemoryStore
│       ├── main.tf					# プロバイダーなど
│       ├── network.tf				# VPC、ロードバランサ、DNS
│       ├── security.tf				# FireWall、SecretManager
│       ├── storage.tf				# CloudStrage					
│       ├── outputs.tf					
│       ├── variables.tf						
│       └── modules/					
│           ├── cloudrun_job/		# CloudRunJobのモジュール
│           │   ├── main.tf
│           │   ├── output.tf
│           │   └── variables.tf
│           └── cloudrun_service/	# CloudRunServiceのモジュール
│               ├── main.tf
│               ├── output.tf
│               └── variables.tf
│   
├── build.sh						# Dockerイメージ、ビルドファイル作成、push
└── update_passwords.sh				# CloudSQLのパスワード更新
