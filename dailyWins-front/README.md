# Daily Wins 前端

原生 JavaScript 月历和详情面板，已连接 Go 后端：登录、任务持久化、自动保存、失败重试、删除撤销和未完成任务移动。

```powershell
$env:API_TARGET = 'http://127.0.0.1:10006'
npm start
```

打开 http://localhost:5173 。Node.js 18+，无运行时依赖、无需构建；server.mjs 提供静态文件及同源 API 代理。

正式模式不创建示例任务。旧 `daily-wins:v1` 记录原样保留、不上传，页面提供提示和备份下载。草稿与失败编辑在当前页面内保留，关闭或刷新前提示；凭证由 HttpOnly Cookie 管理，不写入 localStorage。

`npm test` 运行业务及保存队列测试，`npm run check` 检查 JS 语法。`tests/browser.cjs` 为 Playwright/Edge 浏览器验收，需要显式 E2E_USERNAME、E2E_PASSWORD 和独立空测试账号。详见项目 docs/local-development.md、docs/api.md、docs/test-results.md。
