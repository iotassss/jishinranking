# 地震ランキング

## app（Lambda 用ビルド）

Lambda の Go runtime (`go1.x`) は非推奨のため、**カスタムランタイム（provided.al2）** としてデプロイする。

### build（Lambda 用バイナリ）

Lambda の実行環境は Amazon Linux2（linux/amd64）のため、
以下のコマンドでビルドし、zip 化する。

```sh
# プロジェクトルートで実行
cd app
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/bootstrap cmd/main.go

# Lambda は "bootstrap" という名前の実行ファイルを要求する
zip -j build/linux/app.zip build/linux/bootstrap

## terraform

### 準備

Terraformは以下の環境変数を自動的に読み取って使用するため、あらかじめ設定しておく。
```sh
export AWS_ACCESS_KEY_ID=xxxx
export AWS_SECRET_ACCESS_KEY=yyyy
export AWS_SESSION_TOKEN=zzzz   # MFA や一時クレデンシャルの場合のみ
export AWS_REGION=ap-northeast-1
export CLOUDFLARE_API_TOKEN=aaaa
export TF_VAR_cloudflare_account_id=bbbb
export TF_VAR_cloudflare_zone_id=cccc
```
対話形式で設定する場合は以下のスクリプトを実行する。

```sh
echo "AWS_ACCESS_KEY_ID:"
read -s ACCESS_KEY

echo "AWS_SECRET_ACCESS_KEY:"
read -s SECRET_KEY

echo "CLOUDFLARE_API_TOKEN:"
read -s CLOUDFLARE_API_TOKEN

echo "TF_VAR_cloudflare_account_id:"
read -s TF_VAR_cloudflare_account_id

echo "TF_VAR_cloudflare_zone_id:"
read -s TF_VAR_cloudflare_zone_id

export AWS_ACCESS_KEY_ID=$ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=$SECRET_KEY
export AWS_REGION=ap-northeast-1
export CLOUDFLARE_API_TOKEN=$CLOUDFLARE_API_TOKEN
export TF_VAR_cloudflare_account_id=$TF_VAR_cloudflare_account_id
export TF_VAR_cloudflare_zone_id=$TF_VAR_cloudflare_zone_id
export TF_VAR_worker_aws_access_key_id="$AWS_ACCESS_KEY_ID"
export TF_VAR_worker_aws_secret_access_key="$AWS_SECRET_ACCESS_KEY"
```
※ 入力内容は画面に表示されません
※ この設定は現在のターミナルセッションでのみ有効です

### 基本的な操作

```sh
# 初期化
terraform init
# 計画確認
terraform plan
# 適用
terraform apply
# 状態確認
terraform state list
```

### 注意点
個人運用のためロックは不要
ただし同時に複数ターミナルやCIから terraform apply を走らせないよう注意

### メモ
```sh
# S3バケット削除時
terraform state rm aws_s3_bucket.jishinranking_data
# 全削除
terraform destroy
```

```sh
# ローカル実行
DATA_BUCKET="jishinranking-com" HTML_KEY="public/html/index.html" go run app/cmd/main.go
```

## S3モック
```sh
docker run -d --name minio \
  -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  -v "$PWD/minio-data:/data" \
  minio/minio server /data --console-address ":9001"
```

## TODO
- cloudflare workersから非公開s3バケットにアクセスできるようにする
