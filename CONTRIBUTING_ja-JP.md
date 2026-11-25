# Gopeed コントリビューターガイド

まず最初に、Gopeed への貢献に興味を持っていただきありがとうございます。このガイドは、あなたが Gopeed の
開発に参加するための手助けとなるでしょう。

## ブランチ

Gopeed をフォークし、最新の `main` から作業ブランチを作成して、`main` 宛てにプルリクエストを送ってください。

## ローカル開発

CI と同じ Go 1.27.4 と Flutter 3.41.6（Dart 3.11.4）を使用してください。Web で動作を確認する場合は、リポジトリのルートから Go の REST API を起動します。

```sh
go run -tags nosqlite ./cmd/api
```

API は `127.0.0.1:9999` で待ち受けます。別のターミナルでリポジトリのルートを開き、Chrome で Flutter アプリを実行します。

```sh
cd ui/flutter
flutter pub get
flutter gen-l10n
flutter run -d chrome
```

デスクトップ版やモバイル版を開発する場合は、対象プラットフォーム向けの Go バインディングをビルドします。手順は [README](README_ja-JP.md) を参照してください。

## 翻訳

Flutter の翻訳ファイルは `ui/flutter/lib/l10n` にあります。`app_en.arb` を原文テンプレートとして、
`app_<locale>.arb`（例: `app_de.arb`、`app_zh_TW.arb`）を編集または追加してください。すべての言語で
英語テンプレートと同じメッセージ key が必要です。

翻訳するのはメッセージ値だけです。`{count}` や `{name}` など、中括弧内の変数名は変更しないでください。
`@` で始まるメタデータ key は翻訳しません。変更した ARB ソースファイルだけをコミットし、
`app_localizations*.dart` は生成もコミットも不要です。PR の CI が全言語を検証し、国際化コードを生成して
統合結果を確認します。

以下の例では、ARB に記述する定義と、アプリが値を渡した後にユーザーが実際に見る表示を示します。

1. メッセージに変数を挿入する

   ARB 定義：

   ```json
   "welcomeUser": "こんにちは、{name}"
   ```

   最終表示：`name = 太郎` の場合は「こんにちは、太郎」と表示されます。語順は変更できますが、`{name}` は残します。

2. 数量に応じて表示を切り替える

   ARB 定義：

   ```json
   "fileCount": "{count, plural, =0{ファイルなし} =1{1 個のファイル} other{{count} 個のファイル}}"
   ```

   最終表示：

   - `count = 0` →「ファイルなし」
   - `count = 1` →「1 個のファイル」
   - `count = 3` →「3 個のファイル」

   `=0`、`=1`、`other` はそれぞれ 0、1、その他の数量を表します。各分岐の中で画面に表示される文字だけを翻訳します。

3. 変数の値に応じて表示を切り替える

   ARB 定義：

   ```json
   "taskState": "{state, select, running{実行中} paused{一時停止} other{不明}}"
   ```

   最終表示：

   - `state = running` →「実行中」
   - `state = paused` →「一時停止」
   - その他の値 →「不明」

   `running`、`paused`、`other` はアプリが渡す分岐値なので翻訳せず、その後の中括弧内にある表示文字だけを翻訳します。

## コードの確認

変更した Go ファイルを `gofmt` で整形し、リポジトリのルートから Go のテストを実行します。

```sh
go test -tags nosqlite ./...
```

変更した Dart ファイルを `dart format` で整形します。上記の手順で Flutter の依存関係を取得し、ローカライズコードを生成した後、リポジトリのルートから以下のチェックを実行します。

```sh
cd ui/flutter
flutter analyze
flutter test
```

`lib/api/model/` のアノテーション付きモデルを変更した場合は、`ui/flutter` から JSON シリアライザーを再生成します。

```sh
dart run build_runner build
```
