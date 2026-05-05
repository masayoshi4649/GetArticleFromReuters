package main

import "time"

// sectionResponse はReutersのセクション記事一覧APIレスポンスを表します。
type sectionResponse struct {
	Message string `json:"message"`
	Result  struct {
		Articles []articleSummary `json:"articles"`
	} `json:"result"`
}

// articleSummary はセクション記事一覧APIから得られる記事概要を表します。
type articleSummary struct {
	ID            string `json:"id"`
	RevisionID    string `json:"revision_id"`
	Revision      string `json:"revision"`
	Title         string `json:"title"`
	BasicHeadline string `json:"basic_headline"`
	Description   string `json:"description"`
	CanonicalURL  string `json:"canonical_url"`
	DisplayTime   string `json:"display_time"`
	UpdatedTime   string `json:"updated_time"`
}

// articleDetailResponse はReutersの記事詳細APIレスポンスを表します。
type articleDetailResponse struct {
	StatusCode int           `json:"statusCode"`
	Message    string        `json:"message"`
	Result     articleDetail `json:"result"`
}

// articleDetail はReutersの記事詳細本文を含む記事情報を表します。
type articleDetail struct {
	ID              string           `json:"id"`
	RevisionID      string           `json:"revision_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	CanonicalURL    string           `json:"canonical_url"`
	PublishedTime   string           `json:"published_time"`
	DisplayTime     string           `json:"display_time"`
	UpdatedTime     string           `json:"updated_time"`
	Dateline        []string         `json:"dateline"`
	ContentElements []contentElement `json:"content_elements"`
}

// contentElement はReutersの記事本文を構成する個別要素を表します。
type contentElement struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// appConfig はTOML設定ファイル全体の構造を表します。
type appConfig struct {
	System   systemConfig   `toml:"system"`
	Postgres postgresConfig `toml:"postgres"`
}

// systemConfig はアプリケーション動作に関する設定を表します。
type systemConfig struct {
	Size int `toml:"size"`
}

// postgresConfig はPostgreSQL接続に必要な設定を表します。
type postgresConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
}

// newsArticleRecord はnews_articlesテーブルへ保存する記事レコードを表します。
type newsArticleRecord struct {
	Provider    string
	ArticleID   string
	RevisionID  string
	PublishedAt time.Time
	UpdatedAt   any
	Headline    string
	BodyText    string
}
