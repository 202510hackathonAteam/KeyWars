#!/bin/bash
ZONE="us-central1-a"
INSTANCE="bastion-vm"

# 一時パブリックIP追加
gcloud compute instances add-access-config $INSTANCE --zone $ZONE

# SSH でパッケージインストール
gcloud compute ssh $INSTANCE --zone $ZONE --tunnel-through-iap --command "
  sudo apt update && sudo apt install -y redis-tools
"

# パブリックIP削除
gcloud compute instances delete-access-config $INSTANCE --zone $ZONE --access-config-name "external-nat"