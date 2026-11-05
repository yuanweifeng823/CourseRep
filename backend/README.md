# Backend

后端包含 HTTP 服务入口、健康检查和 PostgreSQL 课程目录查询。课程 API 支持 `q` 搜索（课程代码、名称、教授、专业）及有界分页（`page`、`page_size`）。

```powershell
docker compose up -d postgres
$env:DATABASE_URL = "postgres://courserep:courserep-local@localhost:5433/courserep?sslmode=disable"
./backend/scripts/migrate.ps1 up
go run ./cmd/courserep
```

`GET /healthz` 返回服务健康状态；`GET /api/v1/courses` 从 `courses` 表读取真实课程，空目录时返回空 `items`。例如 `GET /api/v1/courses?q=CS&page=1&page_size=20`。服务启动要求 `DATABASE_URL` 且数据库可连接。

启动本地 PostgreSQL：

```powershell
docker compose up -d postgres
```

迁移使用项目脚本调用固定版本的 goose CLI，不需要全局安装：`./backend/scripts/migrate.ps1 up` 应用迁移，`./backend/scripts/migrate.ps1 status` 查看状态。脚本从 `DATABASE_URL` 读取连接串，迁移文件位于 `migrations/`。`backend/.env.example` 只提供本机 Compose 用法示例；部署环境必须从受控密钥配置注入连接串，不能复用示例凭据。
