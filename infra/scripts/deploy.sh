#!/bin/bash

set -e # エラーで停止

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 確認プロンプト
confirm_step() {
  local step_name=$1 # 1つ目の引数
  local default=${2:-yes}  # 2つ目の引数
  
  if [ "$default" = "yes" ]; then
    read -p "Run ${step_name}? (Y/n): " response
    response=${response:-yes}
  else
    read -p "Run ${step_name}? (y/N): " response
    response=${response:-no}
  fi
  
  [ "$response" = "yes" ] || [ "$response" = "y" ] # Yesかyならtrue
}

echo "[1/7] Build Phase"
if confirm_step "Build"; then
  bash "${SCRIPT_DIR}/build.sh"
  echo "Build completed"
else
  echo "Skipped"
fi
echo ""

echo "[2/7] Push Phase"
if confirm_step "Push"; then
  bash "${SCRIPT_DIR}/push.sh"
  echo "Push completed"
else
  echo "Skipped"
fi
echo ""

echo "[3/7] Firebase Deploy"
if confirm_step "Firebase Deploy"; then
  cd "${INFRA_DIR}/firebase"
  firebase deploy
  echo "Firebase deploy completed"
else
  echo "Skipped"
fi
echo ""

echo "[4/7] Terraform Apply"
if confirm_step "Terraform Apply"; then
  cd "${INFRA_DIR}/terraform/service"
  terraform apply -auto-approve
  echo "Terraform apply completed"
else
  echo "Skipped"
fi
echo ""

echo "[5/7] Cloud Run Job"
if confirm_step "Cloud Run Job"; then
  gcloud run jobs execute cloudrun-migration --region=asia-northeast1 --wait
  gcloud run jobs execute cloudrun-seed --region=asia-northeast1 --wait
  echo "Cloud Run Job completed"
else
  echo "Skipped"
fi
echo ""

echo "[6/7] Update Passwords"
if confirm_step "Password Update" "no"; then
  bash "${SCRIPT_DIR}/update_passwords.sh"
  echo "Passwords updated"
else
  echo "Skipped"
fi
echo ""

echo "[7/7] Bastion Setup"
if confirm_step "Temporary Public IP Task" "no"; then
  bash "${SCRIPT_DIR}/temp_public_ip_task.sh"
  echo "Bastion setup completed"
else
  echo "Skipped"
fi
echo ""

echo "Deploy process finished"