# 2026-03-08 Backend Review Fixes Round 2

## 本次修复范围

本次修复处理以下后端 review 问题：

- `HIGH` `skill_likes` / `skill_favorites` 缺少外键约束，删除 skill 或 user 后可能遗留孤儿记录
- `HIGH` 下载计数 `IncrementDownload` 错误被静默忽略
- `MEDIUM` `LikeSkill` / `UnlikeSkill` 在事务提交后才查询计数，响应存在并发竞态
- `MEDIUM` `name` / `description` / `tags` / `author` / `source_url` 缺少最大长度校验

本次未处理：

- 上传 error path 的文件关闭方式
- `source_url` 的内网地址拦截
- 查询索引优化

## 根因

### 1. engagement 表约束不足

`skill_likes` 和 `skill_favorites` 只有唯一索引，没有数据库级外键约束。业务层即使正常删除资源，也无法保证关联表中的记录被级联清理。

### 2. 下载计数与文件交付耦合不清

下载接口直接调用 `IncrementDownload`，但错误没有记录和暴露，导致计数故障会被吞掉，排障困难。

### 3. 点赞计数读取不在同一事务快照内

`LikeSkill` 和 `UnlikeSkill` 先提交事务，再重新查 `likes_count`。数据库里的累加逻辑本身没问题，但接口返回值可能掺入其他并发请求的结果。

### 4. 文本字段缺少边界控制

上传和更新接口此前只校验必填项，没有对超长文本做统一拦截，可能把异常长输入一路打到数据库或后续处理链路。

## 修复内容

### 数据库

- [db/migrations/0003_add_engagement_foreign_keys.up.sql](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/db/migrations/0003_add_engagement_foreign_keys.up.sql)
  - 先清理 `skill_likes` / `skill_favorites` 中引用不存在 `skill` 或 `user` 的孤儿记录。
  - 为 `skill_id`、`user_id` 增加外键约束。
  - 外键全部使用 `ON DELETE CASCADE`，删除 skill 或 user 时自动清理点赞/收藏记录。
- [db/migrations/0003_add_engagement_foreign_keys.down.sql](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/db/migrations/0003_add_engagement_foreign_keys.down.sql)
  - 提供回滚时移除约束的 down migration。

### 业务与接口

- [backend/internal/service/skill.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/service/skill.go)
  - 保留现有“在旧值上加减”的点赞业务逻辑。
  - `LikeSkill` / `UnlikeSkill` 改为在同一事务内读取更新后的 `likes_count`，再提交事务，避免响应计数读到别的并发请求。
- [backend/internal/handler/skill_read_handlers.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/skill_read_handlers.go)
- [backend/internal/handler/resource_handler.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/resource_handler.go)
  - `/download` 路径改为记录下载计数错误日志，但不阻断文件交付。
  - `/download-hit` 仍保持显式错误返回，方便前端或调用方感知计数失败。

### 输入校验

- [backend/internal/handler/content_validation.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/content_validation.go)
  - 抽出统一文本字段长度校验。
  - 本次采用较宽松上限：
    - `name <= 255`
    - `description <= 5000`
    - `tags <= 1000`
    - `author <= 100`
    - `source_url <= 1024`
- [backend/internal/handler/skill_upload_handlers.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/skill_upload_handlers.go)
- [backend/internal/handler/skill_update_handlers.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/skill_update_handlers.go)
- [backend/internal/handler/resource_handler.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/resource_handler.go)
  - 在 skill / rule / tool / mcp 的创建与更新入口统一接入长度校验。
  - skill 路径返回中文提示，资源通用路径返回英文提示，保持现有接口风格。

## 测试

新增或更新了以下测试：

- [backend/internal/database/engagement_migration_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/database/engagement_migration_test.go)
- [backend/internal/handler/download_count_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/download_count_test.go)
- [backend/internal/handler/input_validation_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/handler/input_validation_test.go)
- [backend/internal/service/skill_like_consistency_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/service/skill_like_consistency_test.go)
- [backend/internal/service/skill_like_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/service/skill_like_test.go)
- [backend/internal/service/skill_favorite_test.go](/Users/qianjianghao/Desktop/Skill_Hub/.worktrees/backend-review-fixes-round2/backend/internal/service/skill_favorite_test.go)

## 验证命令

```bash
cd backend && go test ./internal/database -run EngagementForeignKeyMigration -v
cd backend && go test ./internal/handler -run 'DownloadSkill_Continues|ResourceHandlerDownload_Continues|ValidateContentTextFields|UploadSkill_Rejects|ResourceUpload_Rejects|ReviewedResourceUpdate_Rejects|ResourceUpdate_Rejects' -v
cd backend && go test ./internal/service -run LikeSkill_UsesTransactionScopedCountReader -v
cd backend && go test ./...
cd frontend && npm run build
git diff --check
```

## 上线注意

- `0003` migration 会先删除已有孤儿点赞/收藏记录，再添加外键；这是有意的数据清洗，不会删除仍然关联到有效 skill / user 的正常数据。
- 新外键生效后，测试和脚本都必须先创建合法 user，再写入 likes / favorites。
- 下载接口现在会把计数失败写入日志；如果后续日志中频繁出现该错误，应继续排查数据库稳定性或计数调用链。
