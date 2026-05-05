CREATE TABLE news_articles (
    provider TEXT NOT NULL,          -- REUTERS / BLOOMBERG

    article_id TEXT NOT NULL,
    revision_id TEXT NOT NULL,

    published_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ,

    headline TEXT NOT NULL,
    body_text TEXT NOT NULL,

    PRIMARY KEY (provider, article_id, revision_id)
);