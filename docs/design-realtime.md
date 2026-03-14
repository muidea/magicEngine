# Realtime Design

## SSE

### 服务端

- `Holder`: 单连接 holder，负责发送事件、心跳和关闭回调
- `HolderRegistry`: 管理多个 holder
- `OnRecv(...)`: 发送单个 SSE 事件
- `heartbeat()`: 发送保活帧
- `EchoSSEID()`: 向客户端回显 SSE ID

### 客户端

- `Client.Get(...)` / `Client.Post(...)`
- 支持：
  - `Accept: text/event-stream`
  - `Last-Event-ID`
  - `retry:` 指令更新重试间隔
  - 最大重试次数

### 当前稳定语义

- 服务端事件帧和心跳帧都包含结束空行
- `Client` 会保留构造时传入的 `retryWait`
- `Client.Get(...)` / `Post(...)` 会继承外部 `context.Context`
- `Holder.Run(...)` 可以安全处理 `nil task`，并会返回 task 的错误

## TCP

### 组件

- `Server`: 接收连接并分发给 `ServerSink`
- `Client`: 主动连接远端
- `Endpoint`: 连接抽象，负责收发数据
- `Observer`: 连接、断开、收包回调
- `SimpleEndpointManger`: 连接表和 observer 分发

### 当前实现语义

- TCP 回调通过 `magicCommon/execute.Execute` 调度
- `OnConnect` / `OnDisConnect` / `OnRecvData` 都可能异步执行
- `Client.SendData(...)` 依赖已建立的 endpoint
- `endpoint.SendData(...)` 现在能正确处理部分写入，不会在分段发送时截断数据

## 适用场景

- 业务事件推送：优先 SSE
- 长连接二进制或自定义协议：优先 TCP
