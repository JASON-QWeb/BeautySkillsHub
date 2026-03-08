# Resource Upload + Detail Verification Matrix

| Resource Type | Upload Route | Upload Modes | Review Flow | Detail Route | Detail Rules |
|---|---|---|---|---|---|
| `skill` | `/resource/skill/upload` | `file`, `folder` | AI review + human review required | `/resource/skill/:id` | 保留现有技能页（安装命令、AI 摘要、人工复核） |
| `rules` | `/resource/rules/upload` | `file(.md/.txt)`, `paste(markdown)` | AI review + human review required | `/resource/rules/:id` | 无安装命令框；展示规则 markdown；保留复核信息 |
| `mcp` | `/resource/mcp/upload` | `metadata` | 无 review，直接发布 | `/resource/mcp/:id` | 标题下展示 GitHub 链接；正文渲染 markdown；支持图片语法 |
| `tools` | `/resource/tools/upload` | `metadata` / `file(archive)` | 无 review，直接发布 | `/resource/tools/:id` | 无安装命令框；正文渲染 markdown；有附件时可下载 |

## API Notes

- `rules` review endpoints:
  - `GET /api/rules/:id/review-status`
  - `POST /api/rules/:id/review/retry`
  - `POST /api/rules/:id/human-review`
- `mcp` upload:
  - requires `upload_mode=metadata`
  - supports `source_url` (`http/https`)
- `tools` upload:
  - supports `upload_mode=metadata` (无附件) or `upload_mode=file` (压缩包附件)
  - archive suffix allowed: `.zip/.tar/.tar.gz/.tgz/.rar/.7z/.xz/.bz2/.gz`
- content images for `mcp/tools` markdown:
  - `POST /api/content-assets/images` (auth required, form field `image`)
  - `GET /api/content-assets/:filename`
