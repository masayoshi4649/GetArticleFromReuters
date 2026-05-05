package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var sectionQueryPayload = map[string]any{
	"arc-site":   "reuters-japan",
	"fetch_type": "collection",
	"offset":     0,
	"section_id": "/economy/",
	"size":       5,
	"website":    "reuters-japan",
}

// buildSectionQueryPayload はReutersセクション記事一覧APIへ渡すquery値の元データを作成します。
//
// 機能:
//   - 共通の記事一覧取得用ペイロードをコピーする
//   - 取得件数をsizeで上書きする
//   - 呼び出し元が安全に変更できる新しいmapを返す
//
// 引数:
//   - size: 取得する記事概要の件数
//
// 返り値:
//   - map[string]any: sizeを反映した記事一覧取得用クエリペイロード
func buildSectionQueryPayload(size int) map[string]any {
	payload := make(map[string]any, len(sectionQueryPayload))
	for key, value := range sectionQueryPayload {
		payload[key] = value
	}
	payload["size"] = size

	return payload
}

// buildArticleQueryPayload は記事IDまたはcanonical URLをもとに、Reuters記事詳細APIへ渡すquery値の元データを作成します。
//
// 機能:
//   - article.IDがある場合はid指定のペイロードを作成する
//   - article.IDがない場合はcanonical URL指定のペイロードを作成する
//   - 記事IDもURLもない場合はエラーを返す
//
// 引数:
//   - article: 詳細取得対象の記事概要
//
// 返り値:
//   - map[string]any: 記事詳細取得用クエリペイロード
//   - error: IDもURLも取得できない場合のエラー。成功時はnil
func buildArticleQueryPayload(article articleSummary) (map[string]any, error) {
	if article.ID != "" {
		return map[string]any{
			"id":      article.ID,
			"website": requestWebsite,
		}, nil
	}

	if article.CanonicalURL != "" {
		return map[string]any{
			"website_url": article.CanonicalURL,
			"website":     requestWebsite,
		}, nil
	}

	return nil, fmt.Errorf("article id / canonical_url が存在しません")
}

// buildRequestParams はReuters APIが要求するJSON文字列入りのURLクエリパラメータを作成します。
//
// 機能:
//   - クエリペイロードをJSON文字列へ変換する
//   - JSON文字列をqueryパラメータへ設定する
//   - URLエンコード前のクエリ値を返す
//
// 引数:
//   - queryPayload: JSONへ変換するクエリペイロード
//
// 返り値:
//   - url.Values: queryパラメータを含むURLクエリ値
//   - error: JSON変換に失敗した場合のエラー。成功時はnil
func buildRequestParams(queryPayload map[string]any) (url.Values, error) {
	query, err := json.Marshal(queryPayload)
	if err != nil {
		return nil, fmt.Errorf("query payload の JSON 変換に失敗しました: %w", err)
	}

	values := url.Values{}
	values.Set("query", string(query))

	return values, nil
}

// buildFetchURL はReutersコンテンツ取得APIへ送信する完全なリクエストURLを組み立てます。
//
// 機能:
//   - クエリペイロードをURLクエリパラメータへ変換する
//   - Reutersコンテンツ取得APIのベースURLと取得元識別子を結合する
//   - エンコード済みクエリ文字列をURLへ付与する
//
// 引数:
//   - source: Reuters APIの取得元識別子
//   - queryPayload: queryパラメータへ変換するペイロード
//
// 返り値:
//   - string: 生成したURL文字列
//   - error: クエリパラメータ生成に失敗した場合のエラー。成功時はnil
func buildFetchURL(source string, queryPayload map[string]any) (string, error) {
	params, err := buildRequestParams(queryPayload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/pf/api/v3/content/fetch/%s?%s", baseHost, source, params.Encode()), nil
}

// buildRequest はReuters APIへ送信するために必要なヘッダーを付与したGETリクエストを作成します。
//
// 機能:
//   - GETメソッドのHTTPリクエストを作成する
//   - User-Agentヘッダーを設定する
//   - Acceptヘッダーを設定する
//
// 引数:
//   - requestURL: リクエスト先の完全なURL
//
// 返り値:
//   - *http.Request: 作成したHTTPリクエスト
//   - error: URL不正などでリクエスト作成に失敗した場合のエラー。成功時はnil
func buildRequest(requestURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTP リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json,text/plain,*/*")

	return req, nil
}

// fetchJSON はReuters APIからJSONレスポンスを取得し、呼び出し元が指定した型へ変換します。
//
// 機能:
//   - 指定URLへのHTTPリクエストを作成する
//   - HTTPクライアントでリクエストを送信する
//   - レスポンスボディを読み取る
//   - HTTPステータスが200 OKであることを確認する
//   - JSONレスポンスを指定された出力先へデコードする
//
// 引数:
//   - client: HTTPリクエストに使用するクライアント
//   - requestURL: 取得先URL
//   - out: JSONデコード先のポインタ
//
// 返り値:
//   - []byte: 読み取ったレスポンスボディ
//   - error: リクエスト、読み取り、ステータス確認、JSON変換で失敗した場合のエラー。成功時はnil
func fetchJSON[T any](client *http.Client, requestURL string, out *T) ([]byte, error) {
	req, err := buildRequest(requestURL)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Reuters API へのリクエストに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンスボディの読み取りに失敗しました: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Reuters API が異常終了しました: status=%s body=%s", resp.Status, string(body))
	}

	if err := json.Unmarshal(body, out); err != nil {
		return nil, fmt.Errorf("JSON のパースに失敗しました: %w", err)
	}

	return body, nil
}

// fetchSectionArticles はReutersの経済セクション記事一覧を取得し、後続処理で扱う記事概要一覧に変換します。
//
// 機能:
//   - セクション記事一覧取得用URLを生成する
//   - 生成したURLを標準出力へ表示する
//   - Reuters APIからセクション記事一覧JSONを取得する
//   - レスポンスから記事概要一覧を取り出す
//
// 引数:
//   - client: HTTPリクエストに使用するクライアント
//   - size: 取得する記事概要の件数
//
// 返り値:
//   - []articleSummary: 取得した記事概要の一覧
//   - error: URL生成またはAPI取得に失敗した場合のエラー。成功時はnil
func fetchSectionArticles(client *http.Client, size int) ([]articleSummary, error) {
	requestURL, err := buildFetchURL(sectionSource, buildSectionQueryPayload(size))
	if err != nil {
		return nil, err
	}

	fmt.Println("[Section Request URL]")
	fmt.Println(requestURL)
	fmt.Println()

	var response sectionResponse
	if _, err := fetchJSON(client, requestURL, &response); err != nil {
		return nil, err
	}

	return response.Result.Articles, nil
}

// fetchArticleDetail は記事概要から詳細取得URLを生成し、Reuters APIから記事本文を含む詳細情報を取得します。
//
// 機能:
//   - 記事詳細取得用ペイロードを生成する
//   - 記事詳細取得用URLを生成する
//   - Reuters APIから記事詳細JSONを取得する
//   - レスポンスから記事詳細を取り出す
//
// 引数:
//   - client: HTTPリクエストに使用するクライアント
//   - article: 詳細取得対象の記事概要
//
// 返り値:
//   - articleDetail: 取得した記事詳細
//   - string: 実際に使用したリクエストURL
//   - error: ペイロード生成、URL生成、API取得で失敗した場合のエラー。成功時はnil
func fetchArticleDetail(client *http.Client, article articleSummary) (articleDetail, string, error) {
	payload, err := buildArticleQueryPayload(article)
	if err != nil {
		return articleDetail{}, "", err
	}

	requestURL, err := buildFetchURL(articleSource, payload)
	if err != nil {
		return articleDetail{}, "", err
	}

	var response articleDetailResponse
	if _, err := fetchJSON(client, requestURL, &response); err != nil {
		return articleDetail{}, "", err
	}

	return response.Result, requestURL, nil
}
