# CourseRep 移动端

## 启动

```powershell
npm install
npm start
```

App 启动时读取 `GET /api/v1/courses`，支持下拉刷新和按课程代码、名称、教授、专业做本地筛选。课程目录为空时会展示空状态。

## API 地址

默认 Android 模拟器地址为 `http://10.0.2.2:8080`，iOS 模拟器/Web 默认用 `http://localhost:8080`。复制 `.env.example` 为 `.env` 可设置 `EXPO_PUBLIC_API_URL`。Android 真机需将地址设为开发机的局域网 IP，并确保防火墙允许 8080 端口。

后端从仓库根目录启动：`go run ./backend/cmd/courserep`，默认监听 8080。
