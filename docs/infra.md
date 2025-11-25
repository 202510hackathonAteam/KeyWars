## ディレクトリ構成

infra/
├──terraform/
 		├── iam/
 		│   ├── main.tf        # 初期IAMユーザーとポリシーを定義
 		│   ├── variables.tf
 		│   └── outputs.tf
 		│
		└── service/
		    ├── main.tf        # プロバイダーなど
		    ├── api.tf         # 有効化しているAPI
		    ├── network.tf     # VPC、ピアリング、ALB
		    ├── database.tf    # CloudSQL、MemoryStore
		    ├── storage.tf     # GCS
		    ├── compute.tf     # CloudRun
		    ├── variables.tf
		    └── outputs.tf
