# Scripts

`/scripts` 现在只保留一个对外入口：

- [local.sh](./local.sh)

它负责本地数据库、迁移、seed、重置数据，以及宿主机开发流的统一操作。旧的拆分脚本已经删除，避免命令继续分裂。

## 常用命令

### 启动本地 PostgreSQL 和 Redis

```bash
./scripts/local.sh db up
```

### 查看数据库容器日志

```bash
./scripts/local.sh db logs
./scripts/local.sh db logs postgres
```

### 停止数据库容器

```bash
./scripts/local.sh db down
./scripts/local.sh db down -v
```

`-v` 会一起删除本地 PostgreSQL / Redis volume。

### 执行 migration

```bash
./scripts/local.sh migrate
```

优先读取 `backend/.env.local` 里的 `DATABASE_URL`；如果没有该文件，则自动回退到本地默认 PostgreSQL：

```text
postgres://skillhub:skillhub@localhost:5432/skillhub_local?sslmode=disable
```

### 应用本地 seed

```bash
./scripts/local.sh seed
```

默认 seed 文件是 [db/seed/local_seed.sql](../db/seed/local_seed.sql)，也可以用 `SEED_FILE=/path/to/file.sql` 覆盖。

### 清空业务数据和本地资源文件

```bash
./scripts/local.sh reset
```

这个命令会：

- 调用 [backend/cmd/clear-db](../backend/cmd/clear-db)
- 清空业务表数据
- 清空 `backend/uploads`、`backend/thumbnails`、`backend/avatars`

### 启动宿主机开发流

```bash
./scripts/local.sh dev
```

它会按顺序执行：

1. 启动 Docker PostgreSQL / Redis
2. 执行 migration
3. 默认执行 seed
4. 启动后端进程
5. 启动前端 Vite 开发服务器

如果不想自动 seed：

```bash
SEED_LOCAL=0 ./scripts/local.sh dev
```

## 环境变量

最常用的覆盖项：

- `DATABASE_URL`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `POSTGRES_PORT`
- `PORT`
- `FRONTEND_PORT`
- `SEED_LOCAL`
- `SEED_FILE`

## 设计原则

- 本地开发尽量零配置可用
- 优先复用 `backend/.env.local`
- 没有 `backend/.env.local` 时自动回退到本地默认值
- 脚本只解决“本地运行和维护”；公开仓库的启动入口以 [README.md](../README.md) 和 [AIREAD.md](../AIREAD.md) 为准
