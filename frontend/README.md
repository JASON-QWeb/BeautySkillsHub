# Frontend

`/frontend` 是 Skill Hub 的 React + Vite 前端，负责：

- 首页、详情页、资料页和上传页
- `skill / rules / mcp / tools` 四类资源的展示与编辑
- 登录态管理、401 会话失效处理
- 前端轻量回归测试和 Docker 静态站点构建

## 目录结构

- [src/components](./src/components)
  - 通用 UI 组件
- [src/contexts](./src/contexts)
  - 登录态、对话框等跨页面状态
- [src/features](./src/features)
  - 按业务拆分的页面与功能模块
- [src/services/api](./src/services/api)
  - API 请求层与类型
- [scripts/run-node-tests.mjs](./scripts/run-node-tests.mjs)
  - 前端轻量 node 测试入口
- [Dockerfile](./Dockerfile)
  - 生产静态站点镜像
- [nginx.conf](./nginx.conf)
  - 前端容器内 Nginx 配置

## 本地运行

```bash
cd frontend
npm ci
npm run dev
```

默认开发地址：

- `http://localhost:5173`

默认 API 代理目标：

- `http://localhost:8080`

可通过 [frontend/.env.local.example](./.env.local.example) 中的 `VITE_API_BASE_URL` 覆盖。

## 测试与构建

### 轻量 node 测试

```bash
cd frontend
npm run test:node
```

这套测试覆盖：

- API 请求层
- README 缓存
- Dialog Escape 键逻辑
- AI 角色鼠标调度
- profile 活动文案
- 上传标签序列化
- Docker / CI 回归检查

### 生产构建

```bash
cd frontend
npm run build
```

## Docker

构建镜像：

```bash
docker build -f frontend/Dockerfile frontend
```

当前镜像特性：

- 使用 `nginxinc/nginx-unprivileged:alpine`
- 容器内监听 `8080`
- 以非 root 用户运行

## 关键文件

- [src/App.tsx](./src/App.tsx)
- [src/main.tsx](./src/main.tsx)
- [src/services/api/request.ts](./src/services/api/request.ts)
- [docker-runtime.test.mjs](./docker-runtime.test.mjs)
- [vite.config.ts](./vite.config.ts)
