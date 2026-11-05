# Backend

后端包含 HTTP 服务入口、健康检查和 PostgreSQL 课程目录查询。课程 API 支持 `q` 搜索（课程代码、名称、教授、专业）、`department`/`semester` 精确筛选、按 `rating`/`reviews`/`posts` 排序及有界分页（`page`、`page_size`）；`GET /api/v1/course-facets` 返回当前目录的专业和学期筛选项。

```powershell
docker compose up -d postgres
$env:DATABASE_URL = "postgres://courserep:courserep-local@localhost:5433/courserep?sslmode=disable"
./backend/scripts/migrate.ps1 up
go run ./cmd/courserep
```

`GET /healthz` 返回服务健康状态；`GET /api/v1/courses` 从 `courses` 表读取真实课程，空目录时返回空 `items`。例如 `GET /api/v1/courses?q=CS&page=1&page_size=20`。服务启动要求 `DATABASE_URL` 且数据库可连接。

认证 API：`POST /api/v1/auth/register` 接受 `email`、`password`、`nickname`，向允许的学生邮箱域名发送 10 分钟验证码；`STUDENT_EMAIL_DOMAINS` 支持逗号分隔配置，默认仅 `my.cityu.edu.hk`。`POST /api/v1/auth/verify` 接受 `email`、`code` 并创建已验证账号。`POST /api/v1/auth/login` 登录，返回 15 分钟访问令牌和 30 天刷新令牌；`POST /api/v1/auth/refresh` 轮换刷新令牌；`POST /api/v1/auth/logout` 撤销刷新会话；`GET /api/v1/auth/me` 检查访问令牌和账号状态。认证需配置至少 32 字节随机 `AUTH_SECRET`。邮件配置使用 `SMTP_HOST`、`SMTP_PORT`、`SMTP_FROM`，可选 `SMTP_USER`、`SMTP_PASSWORD`。验证码、密码和刷新令牌只在必要响应中返回，不写日志。注册、验证、登录和刷新现有单进程按 IP 限流；多实例部署前需替换为共享限流存储。

`POST /api/v1/auth/logout` 只撤销请求中传入的 `refresh_token`，不要求访问令牌；令牌缺失或已失效同样返回 204，便于客户端幂等登出。

已验证账号可用 `POST /api/v1/uploads` 以 multipart 字段 `image` 上传 JPEG/PNG（最多 8MB、6000×6000 且 25MP）；图片重编码为 JPEG 并移除 EXIF，文件保存在 `UPLOAD_DIR`（默认 `./uploads`）。响应提供 `/uploads/<随机名>.jpg` 和个人信息遮挡提示；读取图片需带已验证账号访问令牌。图像数据目录不应提交到仓库。

管理员课程导入使用 `POST /api/v1/admin/course-imports/preview` 上传 multipart `file`，支持 UTF-8 CSV/XLSX、最多 5000 行与 10MB；XLSX 读取第一个工作表。可从 [`templates/course-import-template.csv`](templates/course-import-template.csv) 开始填写。表头必须包含 `course_code,course_name,professor,semester,department,credits`（顺序可调整）；除教授外均必填。课程代码最多 40 字符、课程名 200、教授 120、学期 40、专业 120；学分为 0 到 99 的数字。同一文件内课程代码与学期不得重复。响应包含逐行错误及有效记录预览；移动端展示前 30 条有效记录和全部校验错误。没有错误时，使用响应的 `batch_id` 调用 `POST /api/v1/admin/course-imports/{batchID}/confirm` 才会在事务中写入课程目录；相同课程代码/学期作为更新处理。批次 24 小时过期。

普通已验证用户可 `POST /api/v1/reports` 举报公开评价或帖子；运营角色可查看 `/api/v1/admin/reports` 并处理 `/api/v1/admin/reports/{reportID}/decision`。管理员角色可维护 `/api/v1/admin/topic-tags` 词典和执行课程导入。迁移 `00009_seed_topic_tags.sql` 预置 8 个初始话题标签，运营可再增删启停。首个管理员需由受控运维方式在数据库中授予 `users.role='admin'`；不要在公开 API 暴露角色提升能力。

运营角色还可分页查看 `/api/v1/admin/reviews` 和 `/api/v1/admin/posts` 中待审内容，并通过对应的 `/decision` 路由通过或拒绝。待审 DTO 不包含作者关联 ID；匿名帖子仅显示“同校已验证”。

已验证用户可 `POST /api/v1/courses/{courseID}/corrections` 提交课程基础字段纠错；运营角色通过 `/api/v1/admin/course-corrections` 查看队列并调用 `POST /api/v1/admin/course-corrections/{correctionID}/decision` 接受或拒绝。课程代码作为稳定键不可由纠错直接修改。

已验证用户可通过 `POST` / `DELETE /api/v1/courses/{courseID}/completion` 添加或移除已修读课程标记；`GET /api/v1/users/me/courses` 返回本人课程清单。标记数据按用户隔离，不公开。

管理员可在 `/api/v1/admin/community-settings` 设置内容审核处理时限（1 至 720 小时）与社区规范；留空时限表示尚未配置。已验证和校友只读用户可通过 `/api/v1/community-guidelines` 读取，发布页面展示现有设置。数据库初始不预设时限或规范，需由运营方按已确认政策填写。

API 契约可通过运行服务的 `GET /openapi.json` 查看。导出静态 JSON：`go run ./cmd/openapi -out ../mobile/openapi.json`；移动端运行 `npm run generate:api-types` 用 `openapi-typescript` 生成 `mobile/src/api/generated.ts`。产品 HTTP 路由由 Huma 注册并作为外层路由，经 Huma middleware 分发到现有 `http.Handler` 业务实现；运行时路由已统一，但 DTO 校验仍由业务 handler 负责，响应 DTO 和错误响应在 OpenAPI 中仍有宽泛类型，后续可逐端点迁移到 Huma 类型化 handler。

启动本地 PostgreSQL：

```powershell
docker compose up -d postgres
```

迁移使用项目脚本调用固定版本的 goose CLI，不需要全局安装：`./backend/scripts/migrate.ps1 up` 应用迁移，`./backend/scripts/migrate.ps1 status` 查看状态。脚本从 `DATABASE_URL` 读取连接串，迁移文件位于 `migrations/`；脚本以 UTF-8 BOM 保存并合并 goose 的 stderr 输出，可直接在 Windows PowerShell 5.1 下运行。`backend/.env.example` 只提供本机 Compose 用法示例；部署环境必须从受控密钥配置注入连接串，不能复用示例凭据。
