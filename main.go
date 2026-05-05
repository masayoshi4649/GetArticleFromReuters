package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
)

const (
	baseHost         = "https://jp.reuters.com"
	configFileName   = "config.toml"
	sectionSource    = "articles-by-section-alias-or-id-v1"
	articleSource    = "article-by-id-or-url-v1"
	postgresDriver   = "postgres"
	providerReuters  = "REUTERS"
	requestWebsite   = "reuters-japan"
	requestTimeout   = 20 * time.Second
	defaultUserAgent = "Mozilla/5.0"
)

var sectionQueryPayload = map[string]any{
	"arc-site":   "reuters-japan",
	"fetch_type": "collection",
	"offset":     0,
	"section_id": "/economy/",
	"size":       5,
	"website":    "reuters-japan",
}

type sectionResponse struct {
	Message string `json:"message"`
	Result  struct {
		Articles []articleSummary `json:"articles"`
	} `json:"result"`
}

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

type articleDetailResponse struct {
	StatusCode int           `json:"statusCode"`
	Message    string        `json:"message"`
	Result     articleDetail `json:"result"`
}

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

type contentElement struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type appConfig struct {
	System   systemConfig   `toml:"system"`
	Postgres postgresConfig `toml:"postgres"`
}

type systemConfig struct {
	Size int `toml:"size"`
}

type postgresConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
}

type newsArticleRecord struct {
	Provider    string
	ArticleID   string
	RevisionID  string
	PublishedAt time.Time
	UpdatedAt   any
	Headline    string
	BodyText    string
}

func loadConfig(path string) (appConfig, error) {
	var cfg appConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return appConfig{}, fmt.Errorf("config.toml の読み込みに失敗しました: %w", err)
	}

	if err := validateConfig(cfg); err != nil {
		return appConfig{}, err
	}

	return cfg, nil
}

func validateConfig(cfg appConfig) error {
	if cfg.System.Size <= 0 {
		return fmt.Errorf("config.toml の system.size は 1 以上を指定してください")
	}

	if strings.TrimSpace(cfg.Postgres.Host) == "" {
		return fmt.Errorf("config.toml の postgres.host が未設定です")
	}

	if cfg.Postgres.Port <= 0 {
		return fmt.Errorf("config.toml の postgres.port は 1 以上を指定してください")
	}

	if strings.TrimSpace(cfg.Postgres.Username) == "" {
		return fmt.Errorf("config.toml の postgres.username が未設定です")
	}

	if strings.TrimSpace(cfg.Postgres.DBName) == "" {
		return fmt.Errorf("config.toml の postgres.dbname が未設定です")
	}

	return nil
}

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

func buildSectionQueryPayload(size int) map[string]any {
	payload := make(map[string]any, len(sectionQueryPayload))
	for key, value := range sectionQueryPayload {
		payload[key] = value
	}
	payload["size"] = size

	return payload
}

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

func buildRequestParams(queryPayload map[string]any) (url.Values, error) {
	query, err := json.Marshal(queryPayload)
	if err != nil {
		return nil, fmt.Errorf("query payload の JSON 変換に失敗しました: %w", err)
	}

	values := url.Values{}
	values.Set("query", string(query))

	return values, nil
}

func buildFetchURL(source string, queryPayload map[string]any) (string, error) {
	params, err := buildRequestParams(queryPayload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/pf/api/v3/content/fetch/%s?%s", baseHost, source, params.Encode()), nil
}

func buildRequest(requestURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTP リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json,text/plain,*/*")

	return req, nil
}

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

func buildNewsArticleBodyText(detail articleDetail) string {
	parts := make([]string, 0, len(detail.Dateline)+len(detail.ContentElements))
	for _, dateline := range detail.Dateline {
		trimmed := strings.TrimSpace(dateline)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	for _, element := range detail.ContentElements {
		if element.Type != "paragraph" {
			continue
		}

		content := strings.TrimSpace(element.Content)
		if content == "" {
			continue
		}

		parts = append(parts, content)
	}

	return strings.Join(parts, "\n")
}

func parseRequiredTime(fieldName string, value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("%s が未設定です", fieldName)
	}

	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s の日時パースに失敗しました: %w", fieldName, err)
	}

	return parsed, nil
}

func parseOptionalTime(fieldName string, value string) (any, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s の日時パースに失敗しました: %w", fieldName, err)
	}

	return parsed, nil
}

func buildNewsArticleRecord(detail articleDetail) (newsArticleRecord, error) {
	publishedAt, err := parseRequiredTime("published_time", detail.PublishedTime)
	if err != nil {
		return newsArticleRecord{}, err
	}

	updatedAt, err := parseOptionalTime("updated_time", detail.UpdatedTime)
	if err != nil {
		return newsArticleRecord{}, err
	}

	record := newsArticleRecord{
		Provider:    providerReuters,
		ArticleID:   strings.TrimSpace(detail.ID),
		RevisionID:  strings.TrimSpace(detail.RevisionID),
		PublishedAt: publishedAt,
		UpdatedAt:   updatedAt,
		Headline:    strings.TrimSpace(detail.Title),
		BodyText:    buildNewsArticleBodyText(detail),
	}

	if record.ArticleID == "" {
		return newsArticleRecord{}, fmt.Errorf("article_id に対応する result.id が未設定です")
	}

	if record.RevisionID == "" {
		return newsArticleRecord{}, fmt.Errorf("revision_id に対応する result.revision_id が未設定です")
	}

	if record.Headline == "" {
		return newsArticleRecord{}, fmt.Errorf("headline に対応する result.title が未設定です")
	}

	if strings.TrimSpace(record.BodyText) == "" {
		return newsArticleRecord{}, fmt.Errorf("body_text に対応する result.dateline または result.content_elements[type=paragraph].content が未設定です")
	}

	return record, nil
}

func saveArticleToDB(db *sql.DB, detail articleDetail) error {
	record, err := buildNewsArticleRecord(detail)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO news_articles (
			provider,
			article_id,
			revision_id,
			published_at,
			updated_at,
			headline,
			body_text
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.Provider,
		record.ArticleID,
		record.RevisionID,
		record.PublishedAt,
		record.UpdatedAt,
		record.Headline,
		record.BodyText,
	)
	if err != nil {
		return fmt.Errorf("news_articles への INSERT に失敗しました: %w", err)
	}

	return nil
}

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

func articleSummaryRevisionID(article articleSummary) string {
	if revisionID := strings.TrimSpace(article.RevisionID); revisionID != "" {
		return revisionID
	}

	return strings.TrimSpace(article.Revision)
}

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

		if err := saveArticleToDB(db, detail); err != nil {
			return fmt.Errorf("article db の保存に失敗しました: index=%d id=%s err=%w", index, article.ID, err)
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
