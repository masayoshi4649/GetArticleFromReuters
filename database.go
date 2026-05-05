package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

// buildPostgresConnString はPostgreSQL設定からsql.Openへ渡すDSNを組み立てます。
//
// 機能:
//   - ユーザー名とパスワードをURL形式へ設定する
//   - ホスト名とポート番号を接続先として設定する
//   - データベース名をパスとして設定する
//   - sslmode=disableをクエリパラメータへ設定する
//
// 引数:
//   - cfg: PostgreSQLの接続先、認証情報、DB名を含む設定
//
// 返り値:
//   - string: PostgreSQL接続文字列
func buildPostgresConnString(cfg postgresConfig) string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Username, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   "/" + cfg.DBName,
	}

	query := dsn.Query()
	query.Set("sslmode", "disable")
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

// openPostgres はPostgreSQLへの接続を初期化し、Pingで接続可能であることを確認します。
//
// 機能:
//   - PostgreSQL接続文字列を生成する
//   - database/sqlのDBハンドルを初期化する
//   - PingでPostgreSQLへの疎通を確認する
//   - 疎通確認に失敗した場合はDBハンドルを閉じる
//
// 引数:
//   - cfg: PostgreSQL接続に使用する設定
//
// 返り値:
//   - *sql.DB: 接続確認済みのDBハンドル
//   - error: 初期化または疎通確認に失敗した場合のエラー。成功時はnil
func openPostgres(cfg postgresConfig) (*sql.DB, error) {
	db, err := sql.Open(postgresDriver, buildPostgresConnString(cfg))
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL 接続の初期化に失敗しました: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL への接続確認に失敗しました: %w", err)
	}

	return db, nil
}

// saveArticleToDB はReuters記事詳細をDB保存用レコードへ変換し、news_articlesテーブルへINSERTします。
//
// 機能:
//   - Reuters記事詳細からDB保存用レコードを生成する
//   - news_articlesテーブルへ記事レコードをINSERTする
//   - レコード生成またはINSERTに失敗した場合はエラーを返す
//
// 引数:
//   - db: 保存先のDBハンドル
//   - detail: 保存対象のReuters記事詳細
//
// 返り値:
//   - newsArticleRecord: INSERTした記事レコード
//   - error: 保存に失敗した場合のエラー。成功時はnil
func saveArticleToDB(db *sql.DB, detail articleDetail) (newsArticleRecord, error) {
	record, err := buildNewsArticleRecord(detail)
	if err != nil {
		return newsArticleRecord{}, err
	}

	_, err = db.Exec(
		`INSERT INTO news_articles (
			provider,
			article_id,
			revision_id,
			canonical_id,
			published_at,
			updated_at,
			headline,
			body_text
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		record.Provider,
		record.ArticleID,
		record.RevisionID,
		record.CanonicalID,
		record.PublishedAt,
		record.UpdatedAt,
		record.Headline,
		record.BodyText,
	)
	if err != nil {
		return newsArticleRecord{}, fmt.Errorf("news_articles への INSERT に失敗しました: %w", err)
	}

	return record, nil
}

// articleExistsInDB は既に保存済みの記事を重複登録しないように、記事IDとリビジョンIDが一致するReuters記事の存在有無を判定します。
//
// 機能:
//   - 記事概要から記事IDを取得する
//   - 記事概要からリビジョンIDを取得する
//   - IDが不足している場合は未存在として扱う
//   - news_articlesテーブルに一致する記事が存在するか確認する
//
// 引数:
//   - db: 確認先のDBハンドル
//   - article: 確認対象の記事概要
//
// 返り値:
//   - bool: 存在する場合はtrue。存在しない場合、または判定に必要なIDが不足する場合はfalse
//   - error: DB確認に失敗した場合のエラー。成功時はnil
func articleExistsInDB(db *sql.DB, article articleSummary) (bool, error) {
	articleID := strings.TrimSpace(article.ID)
	revisionID := articleSummaryRevisionID(article)
	if articleID == "" || revisionID == "" {
		return false, nil
	}

	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM news_articles
			WHERE provider = $1
				AND article_id = $2
				AND revision_id = $3
		)`,
		providerReuters,
		articleID,
		revisionID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("news_articles の存在確認に失敗しました: %w", err)
	}

	return exists, nil
}
