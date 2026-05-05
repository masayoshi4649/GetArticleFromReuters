| DBカラム       | Reuters                                                                 | Bloomberg                                               |
| -------------- | ----------------------------------------------------------------------- | ------------------------------------------------------- |
| `provider`     | 固定値 `REUTERS`                                                        | 固定値 `BLOOMBERG`                                      |
| `article_id`   | `result.id`                                                             | `[0].id`                                                |
| `revision_id`  | `result.revision_id`                                                    | `[0].revision`                                          |
| `published_at` | `result.published_time`                                                 | `[0].storyPublishedAt` 優先、なければ `[0].publishedAt` |
| `updated_at`   | `result.updated_time`                                                   | `[0].storyUpdatedAt` 優先、なければ `[0].updatedAt`     |
| `headline`     | `result.title`                                                          | `[0].headlines.plain`                                   |
| `body_text`    | `result.dateline[]` + `result.content_elements[type=paragraph].content` | `[0].body.text` から本文部分だけ抽出                    |
| `canonical_id` | `result.canonical_url`                                                  | `[0].slug`                                              |