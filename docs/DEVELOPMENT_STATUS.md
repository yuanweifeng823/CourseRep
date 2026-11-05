# CourseRep 开发进度与清单

盘点日期：2026-11-05（本轮完成 API 级端到端验收、真实 SMTP 投递、真实课程目录导入与 Android 测试包构建）。状态只依据源码与本机实际执行结果，不把设计稿、依赖缓存或构建产物视为已实现功能。

## 验证方式

- 本机 PostgreSQL 16（Docker Compose，宿主机 5433）+ 空库从零迁移至 version 9；服务以 `HTTP_ADDR=:18080` 运行（8080 落在 Windows 排除端口段内）。
- 本地 SMTP sink（127.0.0.1:1025）用于前期测试；真实外部 SMTP 已用 163 邮箱（465 隐式 TLS）验证：服务端返回 202，邮件实际收到，验证码可完成验证并登录。
- 真实课程目录：`courses/` 下两个 CSV（CIE、EEE）去重后 92 门课程已导入；服务端导入器只接受 `course_code,course_name,professor,semester,department,credits` 六列，源数据的 Category/Remarks 不进入数据库。
- Android 测试包：`mobile/android` 下 `assembleRelease` 构建成功，`EXPO_PUBLIC_API_URL=http://192.168.1.2:18080` 已内联到 JS bundle；产物为 debug 签名，仅用于真机侧载测试。
- 验收脚本放在临时目录、不入库：认证、课程导入（CSV/XLSX）、评价、经验帖、评论/点赞/收藏/通知、举报与运营决定、权限矩阵、OpenAPI 一致性。
- 实际执行：`go build ./...`、`go vet ./...`、`go test ./...`（含 `nonNilStrings`、评分边界、标签去重的回归测试）、移动端 `npx tsc --noEmit`、`scripts/migrate.ps1 up|status` 幂等重跑。

## 已通过的端到端项

- 认证 13 步：注册 202、错码 400、验证 201、登录 200、`/auth/me`、刷新轮换、旧 refresh 复用 401、登出 204、重复注册 409、非白名单域名 400、弱密码 400。
- 真实邮箱投递：`POST /auth/register` 用 `15139334079@163.com` 返回 202，163 邮箱实际收到“CourseRep 邮箱验证码”，验证 201、登录 200。附件/正文为 UTF-8，主题经 MIME Q 编码。服务端为 465/994 端口新增隐式 TLS 发送路径（163 不支持明文 STARTTLS 登录）。
- 迁移：空库应用 00001–00009 至 version 9；重跑 `up` 输出 `no migrations to run`；`scripts/migrate.ps1` 在 Windows PowerShell 5.1 下可直接运行。
- 课程导入：CSV 逐行错误（缺字段、学分非数字、文件内重复）、错误批次拒绝确认、重复导入幂等；XLSX 的 sharedStrings 与 inlineStr 两种写法均可解析；非管理员 403；目录搜索/筛选/分页/排序正确，非法排序 400。
- 真实课程目录：`courses/CIE_Courses.csv`（73 行）与 `courses/EEE_Courses.csv`（63 行）合并去重后 92 门课程，预览 `valid_rows=92`、确认 `imported=92`；`?q=EE5607` 可检索，筛选项出现 8 个部门。源数据无学期与教授，导入时按 `semester=未注明`、`professor=''` 保存，部门按课程代码前缀映射（EE→Electrical Engineering、CS→Computer Science、SYE→Systems Engineering、SM→School of Creative Media、EF→Economics and Finance、MGT→Management、IS→Information Systems、MSE→Materials Science and Engineering）。
- 评价：评分范围、未知标签、超长正文、重复提交 409、未审核不公开、管理员通过/拒绝、审核通知落库、审核参数错误 400、他人编辑 404。驳回后需作者编辑并重新送审（当前设计，不视为缺陷）。
- 经验帖：图片上传（仅 JPEG/PNG、重编码去 EXIF、返回遮挡提示）、标题/正文/年份/署名/图片数量校验、未审核不公开、发布后按年份（含“年份未注明”）、话题、最新/热度筛选、匿名显示“同校已验证”、一层评论、点赞/收藏幂等且撤销 204、编辑重审、他人编辑 404。
- 举报与运营：类别与目标校验、待审队列匿名遮罩、驳回/下架内容/停用作者三类决定、下架后内容 404、停用后登录与刷新均 401、举报人收到结果通知、运营指标仅运营可见。
- 权限矩阵 78 项：未登录访问业务路由 401；普通用户访问运营路由 403；普通用户浏览与个人路由 200；管理员队列与指标 200。唯一差异：`POST /api/v1/auth/logout` 无令牌也返回 204（只按传入的 `refresh_token` 撤销，幂等设计，OpenAPI 未声明 security）。
- 匿名与隐私：匿名经验帖在详情、列表、评论、通知中均不返回 `author`/`owner`/`email`/`user_id` 字段，展示名固定“同校已验证”；匿名评价公开列表同样无作者字段；运营待审队列（帖子/评价/举报/纠错）不含作者关联字段。
- 契约：`/openapi.json` 为 OpenAPI 3.1.0，44 path / 52 operation 与注册路由一一对应；抽样 GET 响应与 schema 对照无字段漂移；请求 schema 的必填项与运行时一致（`image_urls`、`topic_tags`、`tags`、`display_identity` 均可省略）。
- 健壮性与限流：畸形 JSON、数组请求体、错误 Content-Type 均 400；`GET /auth/login` 405；分页与搜索边界（`page=0`、`page_size=101`、超长 `q`、非数字页码）400；`/posts/abc`、`/posts/-1` 400；不存在的课程返回空帖子列表；`/uploads/{name}` 路径穿越与随机名均 404；向他人通知写已读 404；篡改或伪造访问令牌 401；未审核与已下架的帖子不可点赞/评论；登录连续失利在每分钟第 11 次返回 429，且不影响已签发的 access token。
- 已修课程：标记 200 且出现在 `GET /users/me/courses`，重复标记幂等 200，取消 204 并移出列表，重复取消仍 204，不存在的课程 404，未登录 401。
- 通知已读：`POST /notifications/{id}/read` 204 且 `read_at` 落库，重复调用仍 204（幂等），操作他人通知 404。
- 课程纠错：空字段与未知字段 400（该接口使用 `DisallowUnknownFields`），合法提交 201；普通用户读队列/审核 403；管理员 accept 后课程字段立即更新、纠错人收到 `course_correction_result` 通知，重复审核 404。
- 话题标签与社区设置：创建 201、重名 409、空名 400、普通用户 403；停用后从公开 `GET /topic-tags` 消失但仍在管理列表；改名 200、空 PATCH 400。社区设置 PUT 仅管理员可写，`content_review_sla_hours=0` 400，`GET /community-guidelines` 读到更新值。
- 运营指标：字段与契约一致，且与数据库实查吻合（已发布评价 4、经验帖 6、近 30 天举报 3、近 30 天活跃浏览账号 2，举报率 30% = 3/(4+6)×100）。

## 本轮修复

- 8 处 `jsonb_build_object` 参数补显式类型（`$2::bigint` / `$3::text`）。此前所有通知写入报 `could not determine data type of parameter $2`，在事务内还会连带回滚审核等主操作（返回 500）。
- 新增 `nonNilStrings`：省略的 JSON 数组解码为 nil 时写入空数组，避免违反 `TEXT[] NOT NULL`（发帖/编辑帖、保存评价）。
- 契约修正：帖子 `sort` 枚举由 `recent` 改为实际支持的 `newest`；课程详情 `ratings` 改为显式四维字段；运营指标补 `reports_30d`、`report_rate_definition`、`active_user_definition`；可选字段标注 `required:"false"`；年份补“不得晚于当前年份”的说明。
- 新增 `00009_seed_topic_tags.sql`：8 个初始话题标签。此前话题词典为空，带标签评价/帖子必然 400。
- CSV 模板补一行示例数据。此前仅有表头，导入预览会报“必须包含字段表头和至少一条数据”。
- `scripts/migrate.ps1` 改为 UTF-8 BOM 并合并 stderr 输出，修复 PowerShell 5.1 下的解析失败与 NativeCommandError 噪声。
- 重新生成 `mobile/openapi.json` 与 `mobile/src/api/generated.ts`，移动端类型检查通过。
- 新增 465/994 端口的隐式 TLS 发信路径（`sendMailImplicitTLS`），并在 `SMTP_TLS=implicit` 时可强制启用；主题改用 MIME Q 编码，修复中文主题在部分客户端显示为乱码。

## 开发清单

1. [x] **课程目录后端接入 PostgreSQL**：有界分页、代码/名称/教授/专业搜索、专业/学期筛选、四种排序。
2. [x] **本地数据库迁移与运行流程**：固定版本 goose、PowerShell 脚本、连接示例与运行文档。
3. [~] **移动端 API 与课程发现**：接口与页面已接，Android 测试包已构建；真机走查未做。
4. [x] **学生邮箱注册与验证**：本地 sink 与真实 163 SMTP（465 隐式 TLS）均已验证，含收件、验证、登录。
5. [x] **课程目录导入**：CSV/XLSX、逐行错误与幂等已验；`courses/` 下真实课程表 92 门已导入并可从 API 检索。
6. [x] **课程详情与评价**（API 级）。
7. [~] **经验帖**：API 级已验；真机与真实图片链路未验。
8. [x] **互动与通知**（API 级，通知已随事务落库）。
9. [x] **举报与运营审核**（API 级，含下架与停用作者）。
10. [~] **移动端主要页面与端到端联调**：类型检查通过、测试包已产出；真机走查未做。
11. [x] **接口契约与质量交付**：OpenAPI 与路由、抽样请求/响应核对完成；业务 handler 仍自行校验，Huma 未接管运行时校验。

## 未完成与阻塞

- 真机/模拟器联调与页面走查：测试包已构建（`mobile/android/app/build/outputs/apk/release/app-release.apk`，debug 签名），需同一局域网内安装到 Android 手机后人工走查；手机需访问 `http://192.168.1.2:18080`，Windows 防火墙已按该 exe 路径允许入站。
- 多实例部署的共享限流：当前为单进程内存按 IP 窗口。
- 生产环境 SMTP 发件域名与 SPF/DKIM：当前用个人 163 邮箱直发，正式部署需专用发件服务与域名认证。
- 未知 JSON 字段处理不一致：课程纠错接口使用 `DisallowUnknownFields`，帖子/评价等接口静默忽略拼错的字段名（例如 `tags` 代替 `topic_tags`），请求仍返回 2xx 但字段不生效。统一严格校验属行为变更，待确认后再做。
- Huma 类型化业务 handler 迁移：契约已生成，运行时校验仍在 `net/http` handler。

## 产品待确认

- 年份历史下界暂按 1990 实现，运行时上限为当前年份；需求文档未定案。
- 浏览事件保留期限（指标用 `course_view_events`，按用户/课程/日期去重）。
- 正式举报处理时限与社区规范文案（由运营配置，初始为空）。
- 话题词典初始 8 项为占位，需运营确认最终清单。
- `POST /api/v1/auth/logout` 是否要求登录：当前无令牌也返回 204（幂等撤销）。
