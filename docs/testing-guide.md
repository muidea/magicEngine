# Testing Guide

## 推荐命令

### 全量

```bash
GOCACHE=/tmp/magicengine-gocache go test ./... -count 1
```

### HTTP

```bash
GOCACHE=/tmp/magicengine-gocache go test ./http -count 1
```

### SSE

```bash
GOCACHE=/tmp/magicengine-gocache go test ./sse -count 1
```

### TCP

```bash
GOCACHE=/tmp/magicengine-gocache go test ./tcp -count 1
```

## 当前测试分布

- `http/`: 已有路由、pattern filter、context、response writer、静态服务回归
- `sse/`: 已补客户端 retry/context 和服务端事件帧回归
- `tcp/`: 已补部分写入、写失败断连、收包回调、未连接发送、endpoint manager 生命周期回归

## 环境说明

- 当前测试不依赖数据库
- 避免在测试里强依赖本机端口监听；受限环境下优先写无网络 direct tests

## 本轮新增回归

- 删除 method 下尾部 route
- 静态目录索引 / fallback
- SSE `retryWait`
- SSE context 取消
- SSE 完整事件帧和心跳帧
- TCP 部分写入和生命周期
