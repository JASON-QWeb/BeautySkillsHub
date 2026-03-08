# Root Docs Refresh Design

## Goal

把根目录面向维护者的文档入口刷新到和当前代码一致，避免继续传播旧的开发、部署和 AI 审核信息。

## Scope

本次同步覆盖：

- [README.md](/Users/qianjianghao/Desktop/Skill_Hub/README.md)
- [DEVELOPMENT.md](/Users/qianjianghao/Desktop/Skill_Hub/DEVELOPMENT.md)
- [ARCHITECTURE.md](/Users/qianjianghao/Desktop/Skill_Hub/ARCHITECTURE.md)
- [DEPLOYMENT.md](/Users/qianjianghao/Desktop/Skill_Hub/DEPLOYMENT.md)
- [ai-review流程.md](/Users/qianjianghao/Desktop/Skill_Hub/ai-review流程.md)

## Design

### 1. 文档职责重新切分

- `README.md`
  - 只保留项目概览、快速启动入口、文档地图和关键事实
- `DEVELOPMENT.md`
  - 承接原来 README 里过长的本地开发、脚本、测试、日常维护内容
- `ARCHITECTURE.md`
  - 聚焦系统边界、资源生命周期、核心服务和运维约束
- `DEPLOYMENT.md`
  - 聚焦生产要求、安全约束、镜像/容器部署和验证
- `ai-review流程.md`
  - 单独解释 reviewed resource 与 auto-published resource 的差异

### 2. 以代码事实为准

文档以当前仓库真实实现为准，重点同步：

- PostgreSQL + migration-first
- `/health`
- 非本地环境强制安全 `DATABASE_URL` 与 `JWT_SECRET`
- 安全响应头、CORS allowlist、限流
- 非 root Docker 运行
- `skill/rules` 与 `mcp/tools` 的不同审核/发布流程
- `/api/me/uploads` 带来的 profile 数据获取方式变化
- 结构化日志

### 3. 内容风格

- 用中文写维护文档
- 尽量把“该看哪里”“该跑什么命令”“哪些配置是生产硬约束”写清楚
- 删除过时或危险示例，尤其是生产 `sslmode=disable` 这类内容
