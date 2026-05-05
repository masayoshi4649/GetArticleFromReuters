package main

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// loadConfig はアプリケーション起動に必要な設定をTOMLファイルから読み込み、検証済みの状態にします。
//
// 機能:
//   - 指定されたTOML設定ファイルを読み込む
//   - 読み込んだ設定値をappConfigへデコードする
//   - validateConfigで必須値と値の妥当性を検証する
//
// 引数:
//   - path: 読み込むTOML設定ファイルのパス
//
// 返り値:
//   - appConfig: 検証済みのアプリケーション設定
//   - error: 読み込みまたは検証に失敗した場合のエラー。成功時はnil
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

// validateConfig はアプリケーション設定の必須値と値の妥当性を検証し、設定不備を起動直後に検出します。
//
// 機能:
//   - 取得件数が1以上であることを確認する
//   - PostgreSQLホストが設定されていることを確認する
//   - PostgreSQLポートが1以上であることを確認する
//   - PostgreSQLユーザー名が設定されていることを確認する
//   - PostgreSQLデータベース名が設定されていることを確認する
//   - Discord通知が有効な場合はWebhook URLが設定されていることを確認する
//
// 引数:
//   - cfg: 検証対象のアプリケーション設定
//
// 返り値:
//   - error: 設定が不正な場合のエラー。妥当な場合はnil
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

	if cfg.Discord.Activate && strings.TrimSpace(cfg.Discord.WebhookURL) == "" {
		return fmt.Errorf("config.toml の discord.webhook_url が未設定です")
	}

	return nil
}
