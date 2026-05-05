package main

import "time"

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
