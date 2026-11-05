# CourseRep

面向香港城市大学学生的课程信息与经验分享应用。项目需求、约束和开发入口如下：

- [开发需求文档](CourseRep_开发需求文档.md)：产品范围、用户流程、功能需求和验收要点。
- [仓库开发约束](AGENTS.md)：每项开发任务必须遵循的工作规则和当前选型边界。
- [工程约束索引](docs/constraints/README.md)：按主题查阅产品、架构、API/数据、安全和质量约束。

## 移动端开发

首个 Expo App 切片已建立，当前实现课程发现首页与空目录状态。运行前安装 Node.js 依赖：

```powershell
cd mobile
npm install
npm start
```

可在 Expo Go 或 iOS/Android 模拟器中预览。课程目录目前没有接入后端数据；首版暂不预置课程，因此界面会明确显示空状态。

`backend/` 与根目录 `compose.yaml` 包含 PostgreSQL 课程目录 API 和本地数据库配置。后端迁移与运行步骤见 [后端说明](backend/README.md)。移动端当前接入课程目录 API。
