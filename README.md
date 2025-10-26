# KeyWars

## 開発環境の使い方
### 起動方法
1. 環境変数ファイルの準備  
    `.env.example` を `.env` にコピーし、値が空欄になっている箇所は各自で設定してください。  
    ```bash
    cp .env.example .env
    ```
    ※ .env は個人環境ごとに異なるため、Gitにはコミットしないでください。  

2. 依存関係のインストール  
    初回のみ、フロントエンドの依存パッケージをインストールします。  
    ```bash
    docker compose run --rm frontend yarn install
    ```

3. コンテナの起動  
    全てのサービス（backend / frontend / db など）を立ち上げます。  
    ```bash
    docker compose up
    ```
    > バックグラウンドで動かしたい場合は -d オプションを付けてください。  
    > ```bash
    > docker compose up -d
    > ```

4. 動作確認  
  以下のURLにアクセス。  
  [http://127.0.0.1:8080/hello/](http://127.0.0.1:8080/hello/)  

5. 停止方法  
    ```bash
    docker compose down
    ```