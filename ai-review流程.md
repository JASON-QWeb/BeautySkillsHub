# AI Review 流程

本文档解释当前仓库里 reviewed resource 的审核链路，以及哪些资源不会进入这条链路。

## 1. 哪些资源会进入 AI 审核

### 会进入 reviewed flow

- `skill`
- `rules`

它们的上传与更新都要经过：

```text
upload/update -> AI review -> human review -> publish
```

### 不会进入 reviewed flow

- `mcp`
- `tools`

这两类资源当前是：

- 自动通过
- 自动发布
- 不走 AI 审核
- 不走人工复核

## 2. 首次上传时机

### `skill / rules`

- 用户上传后，后端先保存本地文件和数据库记录
- 记录初始进入 `queued`
- 不会在 AI 审核前直接发布
- `skill` 也不会在上传后立即同步到 GitHub

### `mcp / tools`

- 上传成功后直接写入已发布状态
- 返回给前端时就是 `approved + published`

## 3. 审核目标文件选择

当前 reviewed flow 会从上传内容里抽取关键文件进行逐个审核，重点关注：

- `*.md`
- `*.mdx`
- 脚本文件：`.sh` `.bash` `.zsh` `.ps1` `.bat` `.cmd` `.py` `.js` `.ts` `.mjs` `.cjs`
- 带 shebang 的文件
- 具备可执行位的文件
- 高风险入口文件：
  - `.github/workflows/*.yml`
  - `.github/workflows/*.yaml`
  - `Dockerfile*`
  - `Makefile`
  - `package.json`

## 4. 审核执行阶段

每个资源当前会记录两层状态。

### 4.1 总阶段

- `queued`
- `security`
- `functional`
- `finalizing`
- `done`

### 4.2 文件级状态

- `queued`
- `running`
- `passed`
- `failed`

前端审核页会轮询：

- `GET /api/skills/:id/review-status`
- `GET /api/rules/:id/review-status`

并展示：

- 当前审核文件
- 已完成数 / 总数
- 每个文件的单独状态

## 5. 安全策略

每个目标文件当前会经过两层检查：

### 5.1 本地规则扫描

内置高危命令模式，例如：

- `rm -rf /`
- `curl|wget ... | bash`
- 可疑 `nc -e`
- 根目录危险权限变更
- 磁盘格式化或覆盖命令
- PowerShell 编码执行

### 5.2 AI 语义审核

后端会把资源上下文与目标文件内容交给 AI 审核服务，要求输出结构化结果：

- 是否通过
- 反馈
- 功能摘要
- AI 描述

任一目标文件失败，整个 reviewed flow 就视为不通过。

## 6. OPENAI 缺失时的行为

如果 `OPENAI_API_KEY` 没有配置，当前实现不是“卡住审核”，而是：

- 返回 auto-approved 结果
- `Feedback` 会标记 AI 审核未配置
- `AIDescription` 会写成降级说明

这意味着本地开发环境在没配 OpenAI 时，`skill / rules` 仍然可能通过 AI 审核阶段并继续进入人工复核。

## 7. 重试策略

当前 reviewed flow 支持手动重试：

- 上传者本人可触发
- 默认最多 `3` 次

状态分支：

- 未达上限：`failed_retryable`
- 达到上限：`failed_terminal`

达到上限后：

- 首次上传的 `skill / rules` 需要重新上传
- revision 更新需要重新提交更新

## 8. 人工复核

AI 审核通过后，进入人工复核阶段。

规则：

- 上传者本人不能复核自己的资源
- 只有复核通过后，资源才进入正式发布态

### 对 `skill`

- 人工复核通过后，如已启用 GitHub 同步，会继续触发 GitHub 同步

### 对 `rules`

- 人工复核通过后直接发布
- 不依赖 GitHub 同步

## 9. 更新与 revision

### `skill / rules`

当前更新不是直接覆盖线上版本，而是创建 pending revision：

- revision 自己有独立的 AI 审核状态
- revision 也支持最多 3 次重试
- revision AI + 人工都通过后，才会应用到主资源

### `mcp / tools`

- 更新直接作用在当前资源
- 不创建待审核 revision

## 10. 与 GitHub 同步的关系

当前关系是：

```text
skill upload -> AI review -> human review -> optional GitHub sync
```

因此不会出现：

- 一上传就直接入 GitHub
- AI 未通过但已经同步 GitHub

## 11. 前端展示

前端审核页会展示：

- AI 总状态
- 当前阶段
- 文件级进度
- 安全与功能摘要
- 人工复核动作

相关页面：

- [frontend/src/features/review/ReviewPage.tsx](./frontend/src/features/review/ReviewPage.tsx)
- [frontend/src/features/skill-detail/SkillDetailPage.tsx](./frontend/src/features/skill-detail/SkillDetailPage.tsx)
