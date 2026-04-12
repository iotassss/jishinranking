# MEMO

## appについて（Lambda 用ビルド）

### build（Lambda 用バイナリ & テンプレートファイル）

Lambda の実行環境は Amazon Linux2（linux/amd64）のため、以下のコマンドでビルドし、zip 化する。

**テンプレートファイル（例: `app/internal/template/index.tmpl`）をGoバイナリにembedせず外部ファイルとして参照している場合、Lambdaデプロイ用zip（app.zip）に必ず含めてください。**

#### Goスクリプトでのテンプレートファイル指定例
Lambda実行時はzip内のルートがカレントディレクトリとなるため、Goコードではzip内のパス構成に合わせてファイルパスを指定してください。
```go
tmpl, err := template.ParseFiles("internal/template/index.tmpl")
```
テンプレートファイルをembedしている場合はこの手順は不要です。
