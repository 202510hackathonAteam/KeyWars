#!/bin/bash
set -e

# 環境変数から取得
PROJECT_ID="keywars-477702"
REGION="asia-northeast1"
SQL_INSTANCE="mysql"
SQL_USER="user"
CLOUD_RUN_SERVICES=("cloudrun-api" "cloudrun-websocket")
CLOUD_RUN_JOBS=("cloudrun-migration" "cloudrun-seed")

# Secret名を指定
MYSQL_USER_PASSWORD_SECRET="mysql-user-password"
MYSQL_ROOT_PASSWORD_SECRET="mysql-root-password"


# MySQLユーザーパスワードを設定
update_mysql_user_password() {
    echo "[1/3]MySQL user password update"
    # 新規パスワード生成、SecretManager更新 
    echo -n $(openssl rand -base64 16) | \
        gcloud secrets versions add $MYSQL_USER_PASSWORD_SECRET \
            --project=$PROJECT_ID \
            --data-file=-
    # SecretManagerから取り出し    
    MYSQL_USER_PASSWORD=$(gcloud secrets versions access latest \
    --secret=$MYSQL_USER_PASSWORD_SECRET \
    )
    # CloudSQL更新
    gcloud sql users set-password $SQL_USER \
    --instance=$SQL_INSTANCE \
    --password="$MYSQL_USER_PASSWORD" \
    --project=$PROJECT_ID

    echo "MySQL password updated"
    echo ""
}

# MySQLルートパスワードを設定
update_mysql_root_password() {
    echo "[2/3]MySQL root password update"
    echo -n $(openssl rand -base64 16) | \
            gcloud secrets versions add $MYSQL_ROOT_PASSWORD_SECRET \
                --project=$PROJECT_ID \
                --data-file=-
    MYSQL_ROOT_PASSWORD=$(gcloud secrets versions access latest \
        --secret=$MYSQL_ROOT_PASSWORD_SECRET \
        --project=$PROJECT_ID)

    gcloud sql users set-password root \
        --instance=$SQL_INSTANCE \
        --host=% \
        --password="$MYSQL_ROOT_PASSWORD" \
        --project=$PROJECT_ID

    echo "MySQL root password updated"
    echo ""
}

update_cloudrun() {
    echo "[3/3]CloudRun Service and Job redeploy"
    for service in "${CLOUD_RUN_SERVICES[@]}"; do
        timestamp=$(date -u +%Y%m%d%H%M%S)
        echo "Updating service: $service"
        gcloud run services update "$service" \
            --region=$REGION \
            --project=$PROJECT_ID \
            --update-env-vars="PASSWORD_UPDATED_AT=${timestamp}"
    done

    for job in "${CLOUD_RUN_JOBS[@]}"; do
        timestamp=$(date -u +%Y%m%d%H%M%S)
        echo "Updating job: $job"
        gcloud run jobs update "$job" \
            --region=$REGION \
            --project=$PROJECT_ID \
            --update-env-vars="PASSWORD_UPDATED_AT=${timestamp}"
    done
    echo "CloudRun Service and Job updated"
    echo ""
}


# メイン処理
main() {
    # 対象出力
    echo "Project: $PROJECT_ID"
    echo "SQL Instance: $SQL_INSTANCE"
    echo "SQL User: $SQL_USER"
    echo "Cloud Run Services: ${CLOUD_RUN_SERVICES[@]}"
    echo "Cloud Run Jobs: ${CLOUD_RUN_JOBS[@]}"
    echo ""
    
    update_mysql_user_password
    update_mysql_root_password
    update_cloudrun
    
    echo "completed!"
}

main