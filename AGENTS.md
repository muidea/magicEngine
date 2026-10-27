# magicEngine 开发规则

本文件直接规定本仓的职责、边界和开发要求；单独检出本仓即可阅读执行。仓库内位置均相对于仓库根目录，协作项目通过项目名和 Git 地址识别，不依赖固定工作区或外部规范文件才能理解规则。

## 功能定位与边界

- 提供 HTTP 路由／中间件／静态资源／上传传输／反向代理、SSE、TCP，以及监听、启动错误和有界排空。
- 网络层接入鉴权中间件，但不决定 CAS 角色、订阅、租户策略、应用实例或业务生命周期。文件传输属于本仓，文件归属、受管引用和删除证明属于 File。
- 保留原请求的取消、期限和追踪。协议取消、连接断开和 HTTP 状态不表示业务已提交或已回滚；不增加业务重试／补偿状态。
- 保持公开 HTTP／SSE／TCP 接口兼容，连接管理和通用错误放在本层，业务策略留给调用方。新公共接口必须有实际消费者。

## 协作项目

- magicCommon — https://github.com/muidea/magicCommon.git：通用事件、任务和生命周期。
- magicModulesRepo — https://github.com/muidea/magicModulesRepo.git：装配路由及 CAS Bearer 接入。
- magicFile — https://github.com/muidea/magicFile.git：文件 owner；本仓不管理文件元数据或清理。
- magicRunner — https://github.com/muidea/magicRunner.git：Panel／Gateway／BFF 消费网络能力并决定用例准入。

## 结构、编码与验证

- `http`、`sse`、`tcp` 为正式能力，`example` 为示例；示例不构成额外平台合同。
- Go 工具链以 go.mod 为准，使用 gofmt／goimports、log/slog、显式错误处理。context key 使用自定义类型；避免路由、连接表及关闭过程中的数据竞争。
- 路由修改验证增删、优先级、方法／通配、代理和中间件顺序；生命周期修改验证启动失败、长连接取消和 Shutdown 排空。保留请求的原始业务上下文。
- 根目录执行 `go build ./...`、`go vet ./...`、`go test -count=1 ./...`，先运行最小受影响包，必要时 race；仅使用当前实际存在的工具，不要求不存在的本地 skill／quality-check 脚本。

## 依赖与交付

- 先修改语义 owner，再更新公开合同及直接消费者。工作区内使用 `upgrade_vendor.sh <目标项目...>` 同步依赖；重建前检查工作树和暂存区，保护 vendor 独有修改、原字节及补丁，保护失败即停止。禁止手改 vendor；同步后审查差异，并以默认 vendor 模式验证消费者。本地 replace 模式不能替代此验证；独立检出缺少同步环境时明确记录未验范围。
- 先运行最小受影响测试，再验证直接消费者。仅文档修改不要求格式化未改动的源码或执行部署；代码变化使用 gofmt／goimports、相关测试、go vet 和构建，按风险选择 race／真实数据库验证。源码、vendor、发行包、镜像和 Live 结果分别记录，夹具成功不能替代持久 owner 回读或现场验收。
- 合同或行为变化同时更新本仓使用说明、设计和对应平台材料；本文件的定位和边界直接同步更新，不把必要规则改成外部文档链接。提交按仓库分开使用 Conventional Commits，不提交凭据、密钥、运行数据或构建产物。
