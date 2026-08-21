# magicEngine

`magicEngine` 是一个轻量 Go 网络框架，当前主要包含三类能力：

- HTTP 服务、路由、中间件、静态资源、上传、代理
- SSE 服务端和客户端
- TCP Server / Client / Endpoint
- HTTP / TCP Server 支持启动错误返回和 `Shutdown(context.Context)` 优雅停止

## 目录

- `http/`: HTTP 框架主实现
- `sse/`: Server-Sent Events
- `tcp/`: TCP 接入和连接封装
- `example/`: HTTP / SSE / TCP 示例
- `docs/`: 当前实现说明和测试说明
- `.agents/skills/`: 面向 Codex 的项目 skill

## 先看这些文档

- `docs/design-http.md`
- `docs/design-realtime.md`
- `docs/testing-guide.md`

## 当前实现要点

- HTTP 默认注册 `logger` 和 `recovery` 中间件
- `RouteRegistry` 支持 API version、动态路径参数 `:id` 和通配 `**`
- HTTP 响应记账包装器安全透传 flush/hijack，并可由 `http.ResponseController` 解包；协议升级后不会追加默认 204
- 静态资源支持文件系统和 embed 两种方式
- SSE 支持 holder 注册、心跳、事件推送和客户端重试
- TCP 基于 `magicCommon/execute` 做连接回调调度

## 最近整理过的关键语义

- `RemoveRoute(...)` 现在能正确删除 method 下的尾部路由
- 删除不存在的 route 不会影响同 method 的其他路由
- 代理路由保留重复 query 参数，后端失败统一返回 `502 Bad Gateway`
- `rewriteURL=true` 替换为目标路径，`rewriteURL=false` 按标准反向代理语义拼接目标基础路径和请求路径
- `serveStaticFile(...)` 的目录回退不会再错误复用文件句柄
- SSE 服务端输出现在符合完整事件帧格式，包含结束空行
- SSE 客户端会正确继承调用方 `context.Context`，并保留配置的 `retryWait`

## 测试

```bash
GOCACHE=/tmp/magicengine-gocache go test ./... -count 1
```

详细说明见 `docs/testing-guide.md`。

压测和生产环境应设置 `MAGICENGINE_ENV=production`，关闭 development 模式的逐请求开始/结束日志，仅保留慢请求告警。
