package main

import (
	"fmt"
	"net/http"
	"strings"
)

// run は設定ファイル、PostgreSQL、Reuters APIを連携させ、未保存の記事をDBへ登録します。
//
// 機能:
//   - 設定ファイルを読み込む
//   - PostgreSQL接続を開く
//   - Reutersのセクション記事一覧を取得する
//   - 保存済み記事の詳細取得とDB保存をスキップする
//   - 未保存記事の詳細を取得してDBへ保存する
//   - Discord通知が有効な場合はDB保存後にWebhookへPOSTする
//   - 保存件数とスキップ件数を標準出力へ表示する
//
// 引数:
//   - なし
//
// 返り値:
//   - error: 処理に失敗した場合は原因を含むエラー。成功時はnil
func run() error {
	cfg, err := loadConfig(configFileName)
	if err != nil {
		return err
	}

	db, err := openPostgres(cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close()

	client := &http.Client{Timeout: requestTimeout}

	articles, err := fetchSectionArticles(client, cfg.System.Size)
	if err != nil {
		return err
	}

	fmt.Println("[Articles]")
	fmt.Printf("count: %d\n", len(articles))
	fmt.Println()

	savedCount := 0
	skippedCount := 0
	for index, article := range articles {
		exists, err := articleExistsInDB(db, article)
		if err != nil {
			return fmt.Errorf("article db の存在確認に失敗しました: index=%d id=%s err=%w", index, article.ID, err)
		}

		if exists {
			skippedCount++
			fmt.Printf("[%d/%d]\n", index+1, len(articles))
			fmt.Printf("db_exists: news_articles provider=%s article_id=%s revision_id=%s\n", providerReuters, strings.TrimSpace(article.ID), articleSummaryRevisionID(article))
			fmt.Printf("skipped: article detail fetch\n")
			fmt.Println()
			continue
		}

		detail, requestURL, err := fetchArticleDetail(client, article)
		if err != nil {
			return fmt.Errorf("article detail の取得に失敗しました: index=%d id=%s err=%w", index, article.ID, err)
		}

		record, err := saveArticleToDB(db, detail)
		if err != nil {
			return fmt.Errorf("article db の保存に失敗しました: index=%d id=%s err=%w", index, article.ID, err)
		}

		if err := notifyDiscordArticleSaved(client, cfg.Discord, record); err != nil {
			return fmt.Errorf("discord 通知に失敗しました: index=%d id=%s err=%w", index, article.ID, err)
		}

		fmt.Printf("[%d/%d]\n", index+1, len(articles))
		fmt.Printf("detail_url: %s\n", requestURL)
		fmt.Printf("db_saved: news_articles\n")
		fmt.Println()
		savedCount++
	}

	fmt.Println("[Done]")
	fmt.Printf("saved db rows: %d\n", savedCount)
	fmt.Printf("skipped db rows: %d\n", skippedCount)

	return nil
}
