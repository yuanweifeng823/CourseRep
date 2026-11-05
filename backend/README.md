# Backend

后端包含 HTTP 服务入口、健康检查和 PostgreSQL 课程目录查询。课程 API 支持 `q` 搜索（课程代码、名称、教授、专业）及有界分页（`page`、`page_size`）。

```powershell
docker compose up -d postgres
$env:DATABASE_URL = "postgres://courserep:courserep-local@localhost:5432/courserep?sslmode=disable"
go run ./cmd/courserep
```

`GET /healthz` 返回服务健康状态；`GET /api/v1/courses` 从 `courses` 表读取真实课程，空目录时返回空 `items`。例如 `GET /api/v1/courses?q=CS&page=1&page_size=20`。服务启动要求 `DATABASE_URL` 且数据库可连接。

启动本地 PostgreSQL：

```powershell
docker compose up -d postgres
```

课程表结构位于 `migrations/00001_courses.sql`。自动迁移命令尚未接入；启动 API 前需先在本地数据库应用迁移。Compose 中的凭据仅供本地开发使用，不得用于部署环境。
