# AIREAD

这份文档写给 AI 编码助手。目标只有一个：用最短路径理解仓库并把项目跑起来。

## 1. 先看什么

按这个顺序读取：

1. `README.md`
2. `AIREAD.md`
3. `backend/README.md`
4. `frontend/README.md`
5. `scripts/README.md`
6. `db/SCHEMA.md`

## 2. 最快启动路径

默认优先使用完整本地栈：

```bash
docker compose up -d --build
```

默认地址：

- frontend: `http://localhost:5173`
- backend health: `http://localhost:8080/health`
- backend api: `http://localhost:8080/api/...`

查看状态：

```bash
docker compose ps
docker compose logs -f migrate backend frontend
```

停止：

```bash
docker compose down
```

清空本地卷并停止：

```bash
docker compose down -v
```

## 3. 宿主机开发路径

如果要直接调试源码，优先使用统一脚本入口：

```bash
./scripts/local.sh dev
```

如果要分步执行：

```bash
./scripts/local.sh db up
./scripts/local.sh migrate
./scripts/local.sh seed
```

然后分别启动：

```bash
cd backend && go run ./cmd/server/main.go
cd frontend && npm ci && npm run dev
```

## 4. 关键事实

- 主业务数据库是 PostgreSQL。
- 正式 schema 只来自 `db/migrations/`。
- backend 启动时不会自动建表或改表。
- `skill / rules` 走审核流；`mcp / tools` 走自动发布流。
- 本地 Docker 启动不要求先手动创建 `.env.local`。
- backend 默认端口 `8080`，frontend 默认端口 `5173`。

## 5. 常用验证

```bash
cd backend && go test ./...
cd frontend && npm ci && npm run build
cd frontend && npm run test:node
```

## 6. 仓库结构

- `backend/`: Go API、认证、审核、资源服务
- `frontend/`: React 页面、上传流、详情页、资料页
- `db/`: migrations、seed、schema 文档
- `scripts/`: 本地运行与维护脚本
- `docker-compose.yml`: 最快本地启动入口

## 7. 改动文档时的约束

- 公开仓库优先保留对外可用的说明，不保留个人部署笔记和内部评审记录。
- 需要启动说明时，优先更新 `README.md` 和 `AIREAD.md`。
- 需要子模块细节时，再更新对应目录下的 `README.md`。
