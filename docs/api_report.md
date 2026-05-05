# Reuters API エンドポイント特性レポート

## 全体像

Reuters Japan の記事取得は、以下 2 種類の API を使い分ける構成です。

1. 記事一覧を取る API
2. 単一記事の本文を取る API

どちらも HTTP GET で呼び出し、レスポンスは JSON です。

共通の URL 形式は次の通りです。

```text
https://jp.reuters.com/pf/api/v3/content/fetch/<source>?query=<JSON文字列>&_website=reuters-japan&d=356&mxId=00000000
```

## 記事一覧 API

### エンドポイント

```text
https://jp.reuters.com/pf/api/v3/content/fetch/articles-by-section-alias-or-id-v1
```

### query パラメータ例

```json
{
  "arc-site": "reuters-japan",
  "fetch_type": "collection",
  "offset": 0,
  "section_id": "/economy/",
  "size": 5,
  "website": "reuters-japan"
}
```

### 取得できるもの

- 複数記事の一覧
- 記事 ID
- タイトル
- 見出し
- 要約
- 記事 URL
- 公開時刻
- 更新時刻
- サムネイルや著者などのメタデータ

### 主なレスポンス構造

```json
{
  "message": "Success",
  "result": {
    "articles": [
      {
        "id": "...",
        "title": "...",
        "basic_headline": "...",
        "description": "...",
        "canonical_url": "...",
        "display_time": "...",
        "updated_time": "..."
      }
    ]
  }
}
```

### この API で取れないもの

- 記事本文

## 単一記事詳細 API

### エンドポイント

```text
https://jp.reuters.com/pf/api/v3/content/fetch/article-by-id-or-url-v1
```

### query パラメータ例 1: ID 指定

```json
{
  "id": "DTBH3OS3LFJ2FDINFO57UZUM5I",
  "website": "reuters-japan"
}
```

### query パラメータ例 2: URL 指定

```json
{
  "website_url": "/markets/japan/DTBH3OS3LFJ2FDINFO57UZUM5I-2026-04-21/",
  "website": "reuters-japan"
}
```

### 取得できるもの

- 単一記事のタイトル
- 要約
- 記事 URL
- 公開時刻
- 更新時刻
- 記事本文
- 画像
- taxonomy などの追加メタデータ

### 主なレスポンス構造

```json
{
  "statusCode": 200,
  "message": "Success",
  "result": {
    "id": "...",
    "title": "...",
    "description": "...",
    "canonical_url": "...",
    "display_time": "...",
    "updated_time": "...",
    "content_elements": [
      {
        "type": "paragraph",
        "content": "本文1段落目"
      },
      {
        "type": "paragraph",
        "content": "本文2段落目"
      }
    ]
  }
}
```

### 本文の格納位置

- 本文は `result.content_elements[]` に入る
- 各段落の文字列は `content_elements[].content`
- `type` は `paragraph` になることが多い

## パラメータの意味

### `query`

- 必須
- JSON 文字列を URL エンコードして渡す
- API ごとに中身が変わる

### `_website`

- サイト識別子
- Reuters Japan では `reuters-japan`

### `query.website`

- query 内のサイト識別子
- Reuters Japan では `reuters-japan`

### `d`

- 補助パラメータ
- 例: `356`

### `mxId`

- 補助パラメータ
- 例: `00000000`

## 実装時の使い方

### 記事一覧を取りたい場合

- `articles-by-section-alias-or-id-v1` を使う
- `result.articles[]` を処理する

### 本文を取りたい場合

- `article-by-id-or-url-v1` を使う
- `id` または `website_url` を渡す
- `result.content_elements[]` から本文を組み立てる

## 実装前提

- 一覧取得用 API と本文取得用 API は分かれている
- 本文取得は 1 リクエストで 1 記事
- 実装フローは次の形になる

1. 一覧 API を呼ぶ
2. `result.articles[]` から `id` を取り出す
3. 各 `id` で詳細 API を呼ぶ
4. `content_elements[].content` を連結して本文を作る

## 結論

- 記事一覧は `articles-by-section-alias-or-id-v1`
- 記事本文は `article-by-id-or-url-v1`
- どちらも JSON を返す
- 詳細 API は `id` と `website_url` の両方で呼べる
- 本文は `result.content_elements[].content`
- 取得処理は「一覧取得 → 各記事詳細取得」の 2 段構成で設計する
