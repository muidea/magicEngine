# HTTP Design

## 组件

- `HTTPServer`: 服务入口，负责 middleware 链和 `RouteRegistry` 绑定
- `RouteRegistry`: 路由注册和分发
- `RequestContext`: middleware 链和 route handler 的运行上下文
- `ResponseWriter`: 包装 `http.ResponseWriter`，记录状态码和输出大小

## 启动流程

1. 创建 `RouteRegistry`
2. 注册 route / handler
3. 创建 `HTTPServer`
4. 通过 `Bind(...)` 绑定 registry
5. 通过 `Use(...)` 添加全局 middleware
6. 调用 `Run()`

## 路由规则

- 支持 method: `GET` / `POST` / `PUT` / `DELETE` / `HEAD` / `OPTIONS`
- 支持动态参数：`/demo/:id`
- 支持递归通配：`/api/**`
- 支持 API version 前缀

## 中间件链

- `HTTPServer.Use(...)` 注册全局 middleware
- `RouteRegistry.AddRoute(...)` 可给单路由附加 middleware
- 全局 middleware 先执行，路由 middleware 后执行
- `ctx.Next()` 会继续向下执行
- middleware 或 handler 一旦写响应，链路即停止

## 静态资源

### 文件系统静态资源

- 主入口在 `http/static.go`
- 通过 `StaticOptions` 控制：
  - `RootPath`
  - `PrefixUri`
  - `ExcludeUri`
  - `Fallback`
  - `IndexFile`

### embed 静态资源

- 主入口在 `http/embed_static.go`
- 通过 `NewEmbedStatic(...)` 和 option 配置

## 文件上传

- 通过 `CreateUploadRoute(...)` 创建上传 route
- 默认上传字段名是 `file`
- 可以通过 context 注入：
  - `RelativePath{}`
  - `FileField{}`
  - `FileName{}`

## 最近修复的行为

- `RemoveRoute(...)` 删除末尾 route 时不再误用 `len(s.routes)`
- `serveStaticFile(...)` 在目录索引和 fallback 场景下不会再重复关闭错误文件句柄
