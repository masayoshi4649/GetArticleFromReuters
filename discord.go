package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const discordEmbedColor = 16744448

// discordWebhookPayload はDiscord Webhookへ送信するペイロードを表します。
type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

// discordEmbed はDiscord Webhookのembeds要素を表します。
type discordEmbed struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Color int    `json:"color"`
}

// notifyDiscordArticleSaved はDB保存済みの記事をDiscord Webhookへ通知します。
//
// 機能:
//   - Discord通知設定が無効な場合は何もせず終了する
//   - DB保存済み記事からWebhookペイロードを生成する
//   - 設定されたWebhook URLへJSONをPOSTする
//   - Content-Typeヘッダーにapplication/jsonを設定する
//
// 引数:
//   - client: HTTPリクエストに使用するクライアント
//   - cfg: Discord Webhook通知の設定
//   - record: DBへINSERTした記事レコード
//
// 返り値:
//   - error: Webhook URL未設定、JSON変換、POST、レスポンス異常で失敗した場合のエラー。成功時はnil
func notifyDiscordArticleSaved(client *http.Client, cfg discordConfig, record newsArticleRecord) error {
	if !cfg.Activate {
		return nil
	}

	webhookURL := strings.TrimSpace(cfg.WebhookURL)
	if webhookURL == "" {
		return fmt.Errorf("discord webhook_url が未設定です")
	}

	payload := buildDiscordWebhookPayload(record)
	return postDiscordWebhook(client, webhookURL, payload)
}

// buildDiscordWebhookPayload はDB保存済みの記事からDiscord Webhookペイロードを生成します。
//
// 機能:
//   - docs/discord_wh.jsonのembeds形式に合わせたペイロードを組み立てる
//   - titleへ記事見出しを設定する
//   - urlへReuters記事URLを設定する
//   - colorへ固定色を設定する
//
// 引数:
//   - record: DBへINSERTした記事レコード
//
// 返り値:
//   - discordWebhookPayload: Discord Webhookへ送信するペイロード
func buildDiscordWebhookPayload(record newsArticleRecord) discordWebhookPayload {
	return discordWebhookPayload{
		Embeds: []discordEmbed{
			{
				Title: record.Headline,
				URL:   buildDiscordArticleURL(record.CanonicalID),
				Color: discordEmbedColor,
			},
		},
	}
}

// buildDiscordArticleURL はcanonical IDからDiscord通知に載せるReuters記事URLを生成します。
//
// 機能:
//   - canonical IDの前後空白を除去する
//   - canonical IDが完全なURLの場合はそのまま返す
//   - canonical IDがパスの場合はReutersのホストを付与する
//
// 引数:
//   - canonicalID: DBへ保存したcanonical_id
//
// 返り値:
//   - string: Discord通知に載せるReuters記事URL
func buildDiscordArticleURL(canonicalID string) string {
	trimmed := strings.TrimSpace(canonicalID)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}

	return baseHost + trimmed
}

// postDiscordWebhook はDiscord Webhook URLへJSONペイロードをPOSTします。
//
// 機能:
//   - WebhookペイロードをJSONへ変換する
//   - POSTメソッドのHTTPリクエストを作成する
//   - Content-Typeヘッダーへapplication/jsonを設定する
//   - Discord Webhookへリクエストを送信する
//   - 2xx以外のレスポンスをエラーとして扱う
//
// 引数:
//   - client: HTTPリクエストに使用するクライアント
//   - webhookURL: POST先のDiscord Webhook URL
//   - payload: Discord Webhookへ送信するペイロード
//
// 返り値:
//   - error: JSON変換、リクエスト作成、POST、レスポンス異常で失敗した場合のエラー。成功時はnil
func postDiscordWebhook(client *http.Client, webhookURL string, payload discordWebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord webhook payload の JSON 変換に失敗しました: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord webhook リクエストの作成に失敗しました: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("discord webhook への POST に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook が異常終了しました: status=%s body=%s", resp.Status, string(responseBody))
	}

	return nil
}
