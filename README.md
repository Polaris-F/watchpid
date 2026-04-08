[中文](./README.md) | [English](./README_EN.md)

# watchpid

`watchpid` 是一个面向长任务的轻量 watcher。它的核心目标不是替代 agent，而是把“盯进程是否结束”这件事从 agent 手里接走。任务注册完成后，agent 可以直接脱身，等 `watchpid` 在任务结束时发通知即可。

## 为什么做这个

现在 agent 处理长任务时，常见流程通常是：

- 启动一个长时间运行的命令
- 简单看一下有没有正常启动
- 然后继续等待、轮询，或者口头说“跑完告诉你”

这个职责边界其实是不对的。

`watchpid` 想解决的是：

- 对已经在运行的进程，用 `watchpid watch <pid>`
- 对准备现在启动的命令，用 `watchpid run -- <cmd>`
- 把监听状态持久化到本地
- 在任务结束时通过通知渠道主动提醒

## 当前状态

这个仓库现在是第一版可用骨架。

当前范围：

- Go CLI
- Linux / Windows 双目标编译
- 本地状态持久化
- 先支持 PushPlus
- 命令支持 JSON 输出，方便 agent / skill / MCP 调用

当前已完成：

- CLI 主体结构
- `store / config / process / notify` 分层
- Linux / Windows 编译通过
- 前台执行路径已做本地验证

当前环境限制：

- 在当前沙箱里，后台 `detach` 子进程会被执行环境回收，所以后台 watcher 不能在这里完整验收

## 安装

如果目标机器已经安装 Go，可以直接：

```bash
go install github.com/Polaris-F/watchpid/cmd/watchpid@latest
watchpid version
```

如果不想装 Go，建议直接下载 GitHub Releases 里的预编译包。

更完整的发布与安装说明见：

- [docs/release.md](./docs/release.md)

## 命令模型

```bash
watchpid watch <pid> --name train-exp42 --detach
watchpid run --name train-exp42 --detach -- python train.py --config xxx.yaml
watchpid status <watch_id>
watchpid list
watchpid cancel <watch_id>
watchpid notify test
watchpid notify setup
watchpid notify setup --token <pushplus_token>
watchpid version
```

语义说明：

- `watch <pid>`：监听已经在运行的进程。它能比较可靠地判断“原来的那个进程实例是否结束了”，但一般拿不到可靠退出码。
- `run -- <cmd>`：由 `watchpid` 启动命令，通知质量最高，能拿到 PID、开始时间和退出码。
- `cancel <watch_id>`：当前定义为取消监听，不主动杀目标进程。
- `--json`：给 agent、脚本、skill、MCP 提供机器可读输出。

## 状态目录

默认状态目录：

```text
~/.watchpid/
  config.env
  watches/*.json
  events.jsonl
  logs/*.log
```

也可以通过下面的环境变量覆盖状态目录：

```bash
WATCHPID_HOME=/path/to/state
```

这对沙箱、CI、容器、多实例隔离都很有用。

## 通知配置

配置优先级：

1. 环境变量
2. watch home 下的 `config.env`

当前支持：

```bash
WATCHPID_NOTIFY_CHANNELS=pushplus
WATCHPID_PUSHPLUS_TOKEN=xxxx
```

配置方式：

```bash
watchpid notify setup
watchpid notify setup --token <pushplus_token>
watchpid notify test
```

如果没有配置 token，`watchpid` 会明确提示怎么配置，而不是静默失败。

## 构建

```bash
go build ./...
go build -o watchpid ./cmd/watchpid
make build
make release VERSION=v0.1.0
```

Windows 交叉编译示例：

```bash
GOOS=windows GOARCH=amd64 go build ./...
```

## 对 Agent 的建议用法

1. 任务还没启动时，优先：

```bash
watchpid run --detach --json -- <command> [args...]
```

2. 任务已经在跑、并且用户给了 PID 时，优先：

```bash
watchpid watch <pid> --detach --json
```

3. 注册成功后，agent 应该：

- 明确告诉用户监听已经建立
- 记录返回的 `watch_id`
- 不再继续轮询
- 用户之后再问时，才调用 `watchpid status <watch_id> --json`

更多说明见：

- [docs/agent-integration.md](./docs/agent-integration.md)
- [docs/release.md](./docs/release.md)

## 目录结构

```text
cmd/watchpid/          CLI 入口
internal/cli/          命令解析与输出
internal/watch/        监听生命周期与编排
internal/store/        状态与事件持久化
internal/process/      平台相关的进程探测
internal/notify/       通知抽象与 PushPlus
internal/daemon/       平台相关的后台拉起
internal/model/        共享模型
```

## 后续计划

- 在正常主机上验证后台 `detach` 运行时行为
- 优化通知内容格式
- 增加运行时长和日志尾部摘要
- 在 PushPlus 之后补充更多消息渠道
- 给状态流转和配置解析补测试
