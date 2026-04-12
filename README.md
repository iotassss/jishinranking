# 地震ランキング

## 🎯 Summary
地震ランキングは、地震の規模や影響をランキング形式で表示するサービスです。

### docs/
- にあるドキュメントには、地震ランキングの概要や設計、運用に関する情報が記載されている。

### app/
- 地震ランキングのアプリケーションコードが含まれている。

### terraform/
- AWSリソースのインフラストラクチャコードが含まれている。

## 🚀 本番環境デプロイ

### 1. application build
```sh
make build
```

### 2. リソース反映

```sh
# ~/.aws/credentials, ~/.aws/config のprofileを使用する
export AWS_PROFILE=worker
cd terraform
# 初期化
terraform init
# 計画確認
terraform plan
# 適用
terraform apply
# 状態確認
terraform state list
```

### 3. 即時反映（初回以外の実行）
```sh
aws lambda invoke \
  --function-name jishinranking-batch \
  --log-type Tail \
  --payload '{}' \
  /tmp/lambda_out.json \
  --query 'LogResult' --output text | base64 -d
cat /tmp/lambda_out.json
```

### 3.5. 即時反映（初回実行）
```sh
aws lambda invoke \
  --function-name jishinranking-initialize \
  --log-type Tail \
  --payload '{}' \
  /tmp/lambda_out.json \
  --query 'LogResult' --output text | base64 -d
cat /tmp/lambda_out.json
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

## 💻 ローカル実行

### 1. S3モック起動
```sh
docker run -d --name minio \
  -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  -v "$PWD/minio-data:/data" \
  minio/minio server /data --console-address ":9001"
```

### 2. ローカル環境での動作確認
```sh
# 基本的にこれを実行すれば問題ない
make serve

# テストデータ生成だけ行う場合はこれを実行するが、make serveに内包されているため通常は不要
make gentestdata
# テンプレートファイルをローカルでビルドして出力する場合はこれを実行するが、make serveに内包されているため通常は不要
make makepages
```
