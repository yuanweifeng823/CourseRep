# CourseRep

面向香港城市大学学生的课程信息与经验分享应用，包含课程搜索、结构化评价、经验帖、互动通知和运营审核。

本文用于**从 Git 克隆到本地电脑开发、联调**。PostgreSQL 在 Docker 中运行，Go API 和 Expo 开发服务器在宿主机运行。`docker compose up` 只启动数据库；首次克隆不包含原开发者的账号、课程数据、图片、SMTP 凭据或本地工具。

## 交给 GPT / 其他 agent 部署

使用能访问本地文件并执行终端命令的 agent，在准备存放项目的目录中粘贴下面的提示词。普通网页聊天可以指导，但仍需要自己执行命令。

```text
请把 CourseRep 部署到我的本地电脑，仓库：
https://github.com/yuanweifeng823/CourseRep.git

请实际执行并验证，不要只提供建议：
1. 检查操作系统、shell、Git、Go、Node/npm、Docker Compose 和 Docker 引擎。
   复用现有工具；缺少系统依赖时说明安装方式和需要我完成的操作。
2. 已有克隆时先检查 git status，保留现有改动、配置和数据，否则克隆仓库。
   先读 README.md、AGENTS.md、CourseRep_开发需求文档.md 和
   docs/constraints/README.md，按实际源码和锁文件部署。
3. 按 README 启动 PostgreSQL、等待健康、配置本地环境、应用全部迁移，
   再从 backend 目录启动 API。后端不会自动读取 .env。
   用 npm ci 安装 mobile 锁定依赖，通过 Expo 启动移动端。
4. 本机独立生成并保存 AUTH_SECRET，不输出它。让我在本地私密填写 SMTP
   配置和自己的校内管理员邮箱，不展示或提交已有秘密。
   缺 SMTP 或课程来源时继续完成独立步骤，明确报告剩余阻塞；
   不绕过邮箱验证、不放宽校内邮箱白名单、不伪造课程或验收结果。
5. 根据 Android 模拟器/真机/iOS 模拟器配置 API 地址，检查应用内保存的
   服务器地址覆盖。真机需要同时连通 API 和 Expo 开发服务器。
6. 验证数据库健康、迁移状态、/healthz、/openapi.json，以及未登录访问
   /api/v1/courses 返回 401。条件允许时完成注册、收信、登录和管理员导入。
   执行后端 go test ./... 和移动端 npx tsc --noEmit，报告实际结果。
7. 不修改业务代码掩盖部署错误、不升级依赖或引入额外服务、不覆盖 .env、
   不删除数据卷、不执行 git reset --hard/clean、不提交或推送凭据。
8. 最后给出实际目录、端口、手机 API 地址、启动/停止方法、验证结果、
   未完成项和需要我完成的操作。后台启动进程时记录进程及日志位置并隐藏窗口。

如果尚未说明预览设备，请先完成数据库和后端部署，再确认我要用哪种设备。
```

## 环境准备

主流程以 **Windows 10/11 + PowerShell** 为准，macOS/Linux 见后文补充。每个代码块均说明执行目录；终端 A 保持 API 运行，终端 B 运行 Expo，终端 C 用于检查。

| 工具 | 本项目基线 | 用途 |
| --- | --- | --- |
| Git | 可执行 `git --version` | 克隆、更新 |
| Go | 1.26.2，见 `backend/go.mod` | API、固定版本 goose 迁移 |
| Node.js / npm | 建议 Node.js 24，使用随附 npm | 安装锁定依赖、启动 Expo |
| Docker Desktop | Linux containers，包含 Compose v2；Windows 需启用 WSL 2 等所需功能 | PostgreSQL 16 |
| 预览设备 | Android 真机/模拟器；macOS 也可用 iOS 模拟器 | 运行移动端 |

启动 Docker Desktop，等引擎就绪；安装工具后重新打开终端检查 PATH：

```powershell
git --version
go version
node --version
npm --version
docker version
docker compose version
```

`docker version` 必须有 Server 信息。首次运行需要网络下载 Go/npm 依赖和 PostgreSQL 镜像。无需全局安装 Expo 或 goose，也不需要 Redis、云数据库、隧道或原开发者的 `.tools` 脚本。

移动端以 `mobile/package-lock.json` 为准（Expo SDK 54 / React Native 0.81 / React 19），不要直接更新为最新版本。Expo Go 必须兼容 SDK 54；不兼容时使用匹配客户端或开发构建。Windows 本地可用 Android；iOS 原生构建/模拟器需要 macOS + Xcode。当前依赖未配齐 Web 运行环境，先走原生预览流程。

## 1. 克隆并启动数据库

在准备存放项目的父目录执行；已有克隆请直接进入仓库，先检查本地改动：

```powershell
git clone https://github.com/yuanweifeng823/CourseRep.git
cd CourseRep
git status --short
docker compose up -d postgres
docker compose ps
docker compose exec -T postgres pg_isready -U courserep -d courserep
```

等待 `postgres` 为 `healthy`，且 `pg_isready` 返回 `accepting connections`，再继续。失败时在根目录执行 `docker compose logs --tail 100 postgres`。

默认端口：数据库宿主机 **5433** → 容器 5432，API **8080**，Expo 通常 **8081**（以终端输出为准）。数据库使用命名卷 `courserep-postgres`；Compose 中的口令仅用于本地开发，勿复用到公网环境。

## 2. 创建本地配置

在**仓库根目录、终端 A**执行。已有 `.env` 不会被覆盖；只在 `AUTH_SECRET` 为空时生成随机值，不打印秘密。

```powershell
if (!(Test-Path backend/.env)) { Copy-Item backend/.env.example backend/.env }
if (!(Test-Path mobile/.env)) { Copy-Item mobile/.env.example mobile/.env }
New-Item -ItemType Directory -Force .tools/uploads | Out-Null

$configPath = (Resolve-Path backend/.env).Path
$configText = [IO.File]::ReadAllText($configPath)
if ($configText -match '(?m)^AUTH_SECRET=[ \t]*\r?$') {
    $randomBytes = New-Object byte[] 32
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($randomBytes) } finally { $rng.Dispose() }
    $secretValue = [Convert]::ToBase64String($randomBytes)
    $configText = [regex]::Replace($configText, '(?m)^AUTH_SECRET=[ \t]*\r?$', "AUTH_SECRET=$secretValue")
}
$uploadPath = (Resolve-Path .tools/uploads).Path
$configText = $configText.Replace('UPLOAD_DIR=./uploads', "UPLOAD_DIR=$uploadPath")
[IO.File]::WriteAllText($configPath, $configText, (New-Object Text.UTF8Encoding($false)))
```

用本地编辑器打开 `backend/.env`，核对/补充下表。格式为**每行 `KEY=value`，不加引号、不写 `export`**；不支持多行值或变量插值，值中的 `=`、`#` 无需转义。

| 变量 | 本地配置说明 |
| --- | --- |
| `DATABASE_URL` | 示例已匹配 Compose：`postgres://courserep:courserep-local@localhost:5433/courserep?sslmode=disable` |
| `HTTP_ADDR` | 默认 `:8080`，监听所有网卡供真机访问；占用/保留时可改 `:18080`，同步修改客户端 |
| `AUTH_SECRET` | 上一步生成，至少 32 字节随机值，保留在本地，重启时复用 |
| `STUDENT_EMAIL_DOMAINS` | `my.cityu.edu.hk,cityu.edu.hk`，完整域名精确匹配，不接受子域 |
| `SMTP_HOST` / `SMTP_PORT` | 自己或团队授权使用的 SMTP 主机与端口，向邮件提供方确认 |
| `SMTP_FROM` | 发件地址，需符合 SMTP 服务规则 |
| `SMTP_USER` / `SMTP_PASSWORD` | 服务要求认证时填写登录名与授权码/密码，不粘贴到聊天或提交仓库 |
| `SMTP_TLS` | 可自行新增；465/994 自动走隐式 TLS，其他端口需要隐式 TLS 时填 `implicit`；587 通常留空，使用服务器支持的 STARTTLS |
| `ADMIN_EMAILS` | **自行新增**：填你能接收验证码的校内邮箱，多个用逗号分隔，用于首个管理员初始化 |
| `UPLOAD_DIR` | 上一步将默认值改为 `.tools/uploads` 的绝对路径；已有配置请确认目录可写，换电脑需重新设置路径 |

**SMTP 是完整注册流程的前置条件。** 未配置 SMTP 时 API 仍能启动、通过健康检查，但不能发送验证码。没有默认账号、通用密码或验证绕过方式。学生收件邮箱须在白名单中，SMTP 发件邮箱可为服务允许的其他邮箱。缺少凭据时先完成基础部署，再由负责人私下提供或自行配置发件服务。

`ADMIN_EMAILS` 在注册验证时赋予管理员角色，API 启动时也会提升名单中的已有账号。应明确配置自己的邮箱，避免沿用代码中的项目默认管理员；空值会回退到默认名单。**移出名单不会撤销数据库中已有的管理员角色**，撤权需单独受控处理。

`backend/.env`、`mobile/.env`、`.tools/` 均被 Git 忽略；`EXPO_PUBLIC_*` 会进入客户端包，只能放 API 地址等公开配置。

## 3. 载入配置、迁移、启动 API

仍在**仓库根目录、终端 A**。后端和迁移脚本只读进程环境，**不会自动加载 `backend/.env`**。换终端或修改配置后重新载入；API 需重启才能使用新配置。

```powershell
# 只把配置当文本读取，不执行文件内容，不回显变量值。
Get-Content -LiteralPath backend/.env -Encoding UTF8 | ForEach-Object {
    $line = $_
    if ($line.Trim() -and !$line.TrimStart().StartsWith('#')) {
        $parts = $line -split '=', 2
        if ($parts.Count -ne 2 -or $parts[0] -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') {
            throw '配置格式错误：请使用 KEY=value'
        }
        [Environment]::SetEnvironmentVariable($parts[0], $parts[1], 'Process')
    }
}

Set-Location backend
go mod download
if ($LASTEXITCODE -ne 0) { throw 'Go 依赖下载失败' }
./scripts/migrate.ps1 up
if ($LASTEXITCODE -ne 0) { throw '数据库迁移失败' }
./scripts/migrate.ps1 status
if ($LASTEXITCODE -ne 0) { throw '迁移状态检查失败' }
go run ./cmd/courserep
```

迁移脚本调用固定版本 `goose v3.29.0`，无需全局安装。新库应用 `backend/migrations/` 下全部迁移；当前最高版本为 **10**，重复 `up` 无待执行迁移属于正常结果。看到 `CourseRep API listening` 后保持终端运行。

Go 模块在 `backend/`，不是仓库根目录。下载或迁移失败时先修复，不要跳过迁移。Windows 拦截本地脚本时，可仅对当前可信终端执行 `Set-ExecutionPolicy -Scope Process Bypass` 后重试，不要更改整机策略。

## 4. 验证 API

新开**终端 C，仓库根目录**；若配置的是 18080，修改下方地址：

```powershell
$apiUrl = 'http://localhost:8080'
Invoke-RestMethod "$apiUrl/healthz"
(Invoke-RestMethod "$apiUrl/openapi.json").openapi
curl.exe -s -o NUL -w '%{http_code}' "$apiUrl/api/v1/courses"
```

预期分别为 `status: ok`、`3.1.0`、`401`。课程接口需要登录，401 表示权限校验生效。健康检查不代表 SMTP、注册、导入或手机网络已经验收。

## 5. 配置并启动移动端

新开**终端 B，从仓库根目录进入 `mobile/`**：

```powershell
cd mobile
npm ci
```

安装成功后编辑 `mobile/.env` 的 `EXPO_PUBLIC_API_URL`。地址**不带** `/api/v1`，端口须与 `HTTP_ADDR` 一致：

| 设备 | API 地址示例（端口为 8080 时） |
| --- | --- |
| Android Studio 默认模拟器 | `http://10.0.2.2:8080`，映射到宿主机 |
| 同一局域网的 Android/iOS 真机 | `http://<电脑局域网IPv4>:8080`，例如 `http://192.168.1.20:8080` |
| macOS 的 iOS 模拟器 | `http://localhost:8080` |

真机的 `localhost` 指手机自己。Windows 用 `ipconfig` 查当前 Wi-Fi/以太网 IPv4，不要填 Docker/WSL/VPN 虚拟网卡地址。电脑与手机需网络互通；访客 Wi-Fi 或校园网客户端隔离可能阻止连接。先用手机浏览器访问 `http://<电脑IPv4>:8080/healthz`，确认返回 `{"status":"ok"}`。

```powershell
# 执行目录：mobile/
npm start
```

用兼容 SDK 的 Expo Go 扫码，或启动 Android 模拟器后按 `a`。连接失败时可运行 `npm start -- --lan`，按需允许私有网络上的 API 与 Expo 端口入站，不要整体关闭防火墙。Expo 隧道只解决开发包访问，不会自动转发 Go API。

登录页和「我的」页底部的「服务器地址」可切换 API，保存在设备上，**优先于 `.env` 默认地址**。若仍连到旧演示服务器，先在应用内更新。修改 `.env` 后重启 Expo，必要时 `npm start -- --clear`；已有安装包需要重建默认值，或在应用内修改地址。

## 6. 注册、管理员与课程导入

1. 使用能收信的白名单校内邮箱注册，密码 10–72 字节，昵称 1–40 字。邮件验证码 10 分钟有效，验证后登录。
2. 用 `ADMIN_EMAILS` 中的账号登录，在「我的」进入运营管理。若账号在配置名单前已注册，重启 API 后重新登录。
3. 从 [课程导入模板](backend/templates/course-import-template.csv) 准备 UTF-8 CSV 或 `.xlsx`。**删除虚构示例行**，填写校对后的数据；真实课程源文件不随仓库分发，向团队负责人获取。
4. 管理页面先上传并预览，修复全部错误，再确认写入。字段为 `course_code,course_name,professor,semester,department,credits`，除教授外必填；XLSX 读取首个工作表，最多 5000 行、10MB。相同课程代码 + 学期更新现有记录，详见 [后端说明](backend/README.md)。
5. 普通账号刷新课程页确认搜索可用；有课程后才能选择专业、发布评价或关联课程经验帖。帖子和评价需审核通过才公开。

新库课程、用户和内容均为空是正常状态；迁移只预置初始话题标签，不复制原开发者的数据。缺少真实课程来源时保留空状态，不把模板示例当作正式数据。

## 停止、再次启动与更新

停止 API、Expo：各自终端按 `Ctrl+C`。停止数据库，在**仓库根目录**执行：

```powershell
docker compose stop postgres
```

数据库保存在 Docker 命名卷中，图片保存在 `UPLOAD_DIR`。**不要使用 `docker compose down -v`，它会删除数据库卷。** 重装/清理 Docker 前另行备份数据库和图片。

再次启动：启动 Docker Desktop → 根目录 `docker compose up -d postgres` 等待健康 → 按第 3 节载入配置、检查迁移、运行 API → `mobile/` 执行 `npm start`。无需重新生成密钥或覆盖配置。

更新前妥善保存本地改动，不要强制覆盖：

```powershell
# 执行目录：仓库根目录，工作区已确认可以更新时
git status --short
git pull --ff-only
```

更新后停止旧 API/Expo，按第 3 节下载依赖并应用迁移，`mobile/` 重新执行 `npm ci` 再启动。重要数据先备份；核对新增变量，不覆盖 `.env`。`pull --ff-only` 失败时处理分支差异，不用强制 reset。

## 常见问题

| 现象 | 检查方式 |
| --- | --- |
| Docker pipe / daemon 连接失败 | 启动 Docker Desktop，检查 Linux 引擎和 WSL 2，直到 `docker version` 有 Server 信息 |
| 数据库连接失败 | 根目录查 `docker compose ps` / `logs`，等待 healthy；URL 用宿主机 5433，配置已载入当前终端 |
| `go.mod file not found` | Go 命令从 `backend/` 执行，根目录没有 Go module |
| `.env` 已填却报缺配置 | 按第 3 节重新载入进程环境再启动 |
| 8080 无法监听 | 用 `Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue` 和 `netsh interface ipv4 show excludedportrange protocol=tcp` 检查占用/保留；改 `:18080` 并同步客户端与防火墙，不强杀未知进程 |
| 5433 被占用 | 协调已有数据库，必要时只改 Compose 宿主机映射并同步 `DATABASE_URL`，容器仍为 5432 |
| 注册 503 / 收不到验证码 | 核对 SMTP 主机、端口、发件授权、授权码、TLS 与垃圾箱；健康检查不验证发信 |
| 登录显示认证服务未配置 | 当前进程的 `AUTH_SECRET` 至少 32 字节，修改后重启 API |
| 请求 429 | 按 `Retry-After` 等待，不要循环重试注册 |
| 无运营入口 | 确认 `ADMIN_EMAILS` 已载入、邮箱已验证，重启 API 并重新登录 |
| 课程空 / 首页无推荐 | 管理员先导入；首页需选择专业，分享/高分课程需有审核通过的内容 |
| 手机请求超时 | 先用手机浏览器检查 API 健康地址，再查局域网、防火墙、端口、设备保存的服务器地址 |
| Expo Go SDK 不兼容 | 匹配 SDK 54 客户端或用开发构建，不为部署临时升级 SDK |
| npm.ps1 被策略拦截 | 使用 `npm.cmd` / `npx.cmd`，或只调整可信终端的 Process 策略 |

分享排障信息时只提供脱敏错误，不粘贴 `.env`、完整邮箱、验证码或令牌。当前发信失败路径可能记录收件邮箱与 SMTP 错误，转发日志前也需人工脱敏。

## macOS / Linux 补充

目录、端口和配置规则与主流程相同。下方用 Bash/Zsh，不要直接复制 PowerShell 环境变量语法。完成克隆和 Docker 引擎准备后：

```bash
# 执行目录：仓库根目录，只在文件不存在时复制
test -f backend/.env || cp backend/.env.example backend/.env
test -f mobile/.env || cp mobile/.env.example mobile/.env
mkdir -p .tools/uploads
docker compose up -d postgres
docker compose exec -T postgres pg_isready -U courserep -d courserep
```

用本地编辑器填写 `.env`，包括至少 32 字节安全随机的 `AUTH_SECRET`、本机 `.tools/uploads` 绝对路径、SMTP 和 `ADMIN_EMAILS`。不要将秘密打印给 agent。数据库健康后在同一终端执行；任一步失败先排查再继续：

```bash
# 执行目录：仓库根目录，读取 KEY=value 文本，不 source/eval 文件
while IFS= read -r line || [ -n "$line" ]; do
  line="${line%$'\r'}"
  case "$line" in ''|\#*) continue ;; esac
  key="${line%%=*}"
  if [[ "$line" != *=* || ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
    printf '%s\n' '配置格式错误：请使用 KEY=value' >&2
    break
  fi
  export "$line"
done < backend/.env

cd backend
go mod download
go run -tags='no_mysql no_sqlite3 no_ydb no_azuresql no_clickhouse no_libsql no_mssql no_vertica' github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir ./migrations postgres "$DATABASE_URL" up
go run -tags='no_mysql no_sqlite3 no_ydb no_azuresql no_clickhouse no_libsql no_mssql no_vertica' github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir ./migrations postgres "$DATABASE_URL" status
go run ./cmd/courserep
```

另开终端，在 `mobile/` 执行 `npm ci`，配置设备 API 地址后 `npm start`。用 `curl http://localhost:8080/healthz` 和 `curl -i http://localhost:8080/api/v1/courses` 检查 API，后者未登录应为 401。

## 开发检查与项目文档

以下是可重复执行的检查命令，不代表每次部署都已通过。SMTP 收信、真机网络、图片上传与审核需要实际联调。

```powershell
# 终端 C：从仓库根目录执行
Push-Location backend
go test ./...
go vet ./...
go build ./...
Pop-Location

Push-Location mobile
npx tsc --noEmit
Pop-Location
```

接口变更后，在 `mobile/` 执行 `npm run generate:api-types`；普通部署无需重新生成或手改类型。

| 文档 | 内容 |
| --- | --- |
| [开发需求文档](CourseRep_开发需求文档.md) | 产品范围、用户流程、验收要点 |
| [仓库开发约束](AGENTS.md) / [工程约束索引](docs/constraints/README.md) | 人和 agent 开发前必读 |
| [后端说明](backend/README.md) | API、导入、认证和存储细节 |
| [移动端说明](mobile/README.md) | 页面和客户端行为 |
| [开发进度](docs/DEVELOPMENT_STATUS.md) | 历史验证与未完成项；其中的本机路径、演示域名和账号不作为新电脑部署前提 |

公网生产部署另行规划：生产目标、TLS、专用邮件服务、备份和正式隐私政策仍需确认。
