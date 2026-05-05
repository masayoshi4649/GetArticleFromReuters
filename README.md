# GetArticleFromReuters

## 処理概要
1. 取得条件をもとにクエリペイロードを生成する
2. URL パラメータを組み立てる
3. Reuters API のリクエスト URL を生成する
4. HTTP GET リクエストを作成して JSON を取得する
5. 取得した JSON をarticlesで分解する
6. 単一articleを取得し、 {id}.md という名称で、以下形式にて保存。
````md
## {result.title}
### {result.description}
> DISPLAY {display_time>JST(yyyy-mm-dd HH:MM:SS zzz)}
> UPDATED {updated_time>JST(yyyy-mm-dd HH:MM:SS zzz)}
```
{content_elements.content[0]}
{content_elements.content[1]}
{content_elements.content[2]}
...

{https://jp.reuters.com/ + {result.canonical_url}}
```
````
### 実行方法

```bash
go run .
```

### ビルド方法

```bash
go build .
```
