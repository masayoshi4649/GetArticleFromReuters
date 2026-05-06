# GetArticleFromReuters

Reuters日本語版の経済セクションから記事を取得し、PostgreSQLの`news_articles`テーブルへ保存するWindows向けのバッチ実行プログラムです。新規保存した記事は、設定によりDiscord Webhookへ通知できます。

このREADMEでは、ビルド済みの`getarticlefromreuters.exe`をWindowsタスクスケジューラへ登録し、定期実行する運用を前提に説明します。

## 概要

実行時に次の処理を行います。

1. 作業フォルダ内の`config.toml`を読み込む
2. PostgreSQLへ接続する
3. Reuters日本語版の経済セクション記事一覧を取得する
4. `news_articles`テーブルに未登録の記事だけ詳細本文を取得する
5. 未登録記事を`news_articles`テーブルへ保存する
6. Discord通知が有効な場合、保存した記事をWebhookへ通知する

同一の`provider`、`article_id`、`revision_id`が既にDBに存在する記事はスキップします。

## 前提条件

- Windows環境
- ビルド済みの`getarticlefromreuters.exe`
- PostgreSQLに接続できること
- `news_articles`テーブルが作成済みであること
- インターネットへ接続でき、ReutersおよびDiscord Webhookを利用できること

## 配置するファイル

任意のフォルダを作成し、そのフォルダをこのプログラムの作業フォルダとして使います。

例:

```text
C:\Tools\GetArticleFromReuters\
├─ getarticlefromreuters.exe
└─ config.toml
```

`config.toml`は必ず`getarticlefromreuters.exe`と同じ作業フォルダに置いてください。プログラムはカレントディレクトリから`config.toml`を読み込みます。

## データベース準備

PostgreSQL上に保存先DBを作成し、次のテーブルを作成してください。

```sql
CREATE TABLE news_articles (
    provider TEXT NOT NULL,
    article_id TEXT NOT NULL,
    revision_id TEXT NOT NULL,
    canonical_id TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ,
    headline TEXT NOT NULL,
    body_text TEXT NOT NULL,
    PRIMARY KEY (provider, article_id, revision_id)
);
```

同じSQLは`install/create_news_articles.sql`にもあります。

## 設定ファイル

作業フォルダに`config.toml`を作成します。`config.toml.example`をコピーして編集してください。

```toml
[system]
size = 5

[postgres]
host = "127.0.0.1"
port = 5432
password = "pass123456789"
username = "postgres"
dbname = "newsdb"

[discord]
activate = true
webhook_url = "https://discord.com/api/webhooks/xxxxxxxx/yyyyyyyy"
```

### 設定項目

| セクション | 項目          | 説明                                                                      |
| ---------- | ------------- | ------------------------------------------------------------------------- |
| `system`   | `size`        | 1回の実行でReutersの記事一覧から取得する記事概要件数。1以上を指定します。 |
| `postgres` | `host`        | PostgreSQLのホスト名またはIPアドレス。                                    |
| `postgres` | `port`        | PostgreSQLのポート番号。通常は`5432`です。                                |
| `postgres` | `username`    | PostgreSQLへ接続するユーザー名。                                          |
| `postgres` | `password`    | PostgreSQLへ接続するパスワード。                                          |
| `postgres` | `dbname`      | 保存先データベース名。                                                    |
| `discord`  | `activate`    | Discord通知を行う場合は`true`、行わない場合は`false`。                    |
| `discord`  | `webhook_url` | Discord Webhook URL。`activate = true`の場合は必須です。                  |

## 手動実行での動作確認

タスクスケジューラへ登録する前に、作業フォルダで手動実行して設定とDB接続を確認します。

コマンドプロンプトの例:

```cmd
cd /d C:\Tools\GetArticleFromReuters
getarticlefromreuters.exe
```

正常に動作すると、取得URL、取得件数、DB保存件数、スキップ件数などが標準出力へ表示されます。

エラーが発生した場合は標準エラーへ`error: ...`形式で内容が表示され、終了コード`1`で終了します。

## Windowsタスクスケジューラへの登録

タスクスケジューラのGUIから、利用者の運用に合わせた実行間隔で登録してください。

重要な点は、`getarticlefromreuters.exe`と`config.toml`を置いたフォルダを、タスクの作業フォルダとして指定することです。タスクスケジューラでは、`開始`、または`開始 in`に作業フォルダを指定してください。これを指定しないと、プログラムが`config.toml`を見つけられません。

### GUIで登録する場合

1. Windowsの「タスク スケジューラ」を開く
2. 右側の「基本タスクの作成」または「タスクの作成」を選択する
3. 任意のタスク名を入力する
   - 例: `GetArticleFromReuters`
4. 「トリガー」で実行間隔を設定する
   - 例: 毎日、1時間ごと、30分ごとなど
5. 「操作」で「プログラムの開始」を選択する
6. 次のように設定する

| 項目                  | 設定例                                                     |
| --------------------- | ---------------------------------------------------------- |
| プログラム/スクリプト | `C:\Tools\GetArticleFromReuters\getarticlefromreuters.exe` |
| 引数の追加            | 空欄                                                       |
| 開始                  | `C:\Tools\GetArticleFromReuters`                           |

7. 必要に応じて「ユーザーがログオンしているかどうかにかかわらず実行する」を選択する
8. 保存後、作成したタスクを右クリックして「実行」し、動作確認する

## 運用時の注意点

- `config.toml`にはDBパスワードやWebhook URLが含まれるため、アクセス権限に注意してください。
- `discord.activate = true`の場合、新規保存された記事ごとにDiscord通知が送信されます。
- 既に保存済みの記事はスキップされるため、同じタスクを定期実行しても同一リビジョンの記事は重複登録されません。
- Reuters側の仕様変更や通信エラーにより取得に失敗する場合があります。
- タスクスケジューラで失敗する場合は、まず同じWindowsユーザーで作業フォルダから手動実行してください。

## トラブルシューティング

### `config.toml`の読み込みに失敗する

タスクスケジューラの「開始」が作業フォルダになっているか確認してください。`getarticlefromreuters.exe`と`config.toml`が同じフォルダにあることも確認してください。

### PostgreSQLへの接続に失敗する

`config.toml`の`postgres.host`、`postgres.port`、`postgres.username`、`postgres.password`、`postgres.dbname`を確認してください。また、タスクを実行するWindowsユーザーからPostgreSQLへ接続できることを確認してください。

### `news_articles`へのINSERTに失敗する

保存先DBに`news_articles`テーブルが存在するか確認してください。テーブル定義は`install/create_news_articles.sql`を参照してください。

### Discord通知に失敗する

`discord.activate`が`true`の場合、`discord.webhook_url`が正しいWebhook URLであることを確認してください。通知を使わない場合は`discord.activate = false`にしてください。

## ライセンス

このリポジトリのライセンスは`LICENSE`を参照してください。
