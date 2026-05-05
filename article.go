package main

import (
	"fmt"
	"strings"
	"time"
)

// articleSummaryRevisionID はReuters APIレスポンスの差異を吸収し、DB存在確認に使うリビジョンIDを決定します。
//
// 機能:
//   - revision_idを優先して取得する
//   - revision_idが空の場合はrevisionを代替値として取得する
//   - 取得した値の前後空白を除去する
//
// 引数:
//   - article: リビジョンIDを取得する対象の記事概要
//
// 返り値:
//   - string: revision_idまたはrevision。どちらも空の場合は空文字
func articleSummaryRevisionID(article articleSummary) string {
	if revisionID := strings.TrimSpace(article.RevisionID); revisionID != "" {
		return revisionID
	}

	return strings.TrimSpace(article.Revision)
}

// buildNewsArticleBodyText はReuters記事詳細からDBへ保存する本文テキストを作成します。
//
// 機能:
//   - datelineの空でない要素を本文部品として追加する
//   - content_elementsのparagraph要素だけを本文部品として追加する
//   - 空白のみの本文要素を除外する
//   - 本文部品を改行で連結する
//
// 引数:
//   - detail: 本文生成に使用する記事詳細
//
// 返り値:
//   - string: データラインとparagraph要素の本文を改行で連結した文字列
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

// parseRequiredTime は必須日時項目が空でないことを確認し、RFC3339Nano形式の文字列をtime.Timeへ変換します。
//
// 機能:
//   - 日時文字列の前後空白を除去する
//   - 空文字の場合は未設定エラーを返す
//   - RFC3339Nano形式で日時を解析する
//
// 引数:
//   - fieldName: エラーメッセージに使用する項目名
//   - value: 解析対象の日時文字列
//
// 返り値:
//   - time.Time: 解析済みの日時
//   - error: valueが未設定、または解析に失敗した場合のエラー。成功時はnil
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

// parseOptionalTime は任意日時項目をDBへ渡せる値へ変換し、空値はNULL相当として扱います。
//
// 機能:
//   - 日時文字列の前後空白を除去する
//   - 空文字の場合はnilを返す
//   - 値がある場合はRFC3339Nano形式で日時を解析する
//
// 引数:
//   - fieldName: エラーメッセージに使用する項目名
//   - value: 解析対象の日時文字列
//
// 返り値:
//   - any: valueが空の場合はnil。日時が指定されている場合はtime.Time
//   - error: 日時の解析に失敗した場合のエラー。成功時はnil
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

// buildNewsArticleRecord はReuters APIの記事詳細をnews_articlesテーブルのINSERTに必要な形式へ変換し、必須項目を検証します。
//
// 機能:
//   - published_timeを必須日時として解析する
//   - updated_timeを任意日時として解析する
//   - 記事ID、リビジョンID、見出し、本文を整形する
//   - DB保存に必要な必須項目が存在することを確認する
//
// 引数:
//   - detail: DB保存用レコードへ変換するReuters記事詳細
//
// 返り値:
//   - newsArticleRecord: DB保存用の記事レコード
//   - error: 日時解析または必須項目検証に失敗した場合のエラー。成功時はnil
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
