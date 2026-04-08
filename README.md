# watchpid

`watchpid` is a lightweight watcher for long-running jobs. It lets an agent or human register a running PID or launch a new command, then stop waiting. When the target finishes, `watchpid` is responsible for persisting state and sending a notification.

`watchpid` 是一个面向长任务的轻量 watcher。它的核心目标不是替代 agent，而是把“盯进程是否结束”这件事从 agent 手里接走。任务注册完成后，agent 可以直接脱身，等 `watchpid` 在任务结束时发通知即可。

## Why

Typical agent workflow today:

- start a long command
- check that it looks healthy
- keep waiting, polling, or telling the user “I’ll let you know when it finishes”

That is the wrong ownership boundary.

What `watchpid` tries to do instead:

- `watchpid watch <pid>` for a process that is already running
- `watchpid run -- <cmd>` for a command that should be started now
- persist watcher state locally
- notify through PushPlus first, with more channels added later

今天常见的问题是：agent 启动一个长任务之后，还要一直盯着终端，或者口头说“跑完告诉你”，但实际上仍然得持续等待。

`watchpid` 的目标是把这部分职责单独抽出来：

- 对已经存在的进程，用 `watchpid watch <pid>`
- 对准备现在启动的命令，用 `watchpid run -- <cmd>`
- 把状态持久化到本地
- 任务结束时由通知渠道主动提醒

## Status

This repository is the first usable skeleton.

Current scope:

- Go CLI
- Linux and Windows build targets
- persistent watch state under a local home directory
- PushPlus as the first notification channel
- JSON output for agent-friendly integration

Current implementation status:

- command skeleton is in place
- store/config/process/notifier layers are split
- Linux and Windows builds compile successfully
- foreground flows are locally verified

Known limitation in this environment:

- detached background validation is limited by the current sandbox, which reaps child processes aggressively

当前仓库处于“第一版可用骨架”阶段。

当前范围：

- Go CLI
- Linux / Windows 双目标编译
- 本地状态持久化
- 先支持 PushPlus
- 命令支持 JSON 输出，方便 agent / skill / MCP 调用

当前已完成：

- CLI 主体结构
- store / config / process / notify 分层
- Linux / Windows 编译通过
- 前台执行路径已做本地验证

当前环境限制：

- 在这个沙箱里，后台 detach 子进程会被执行环境回收，所以后台 watcher 不能在这里完整验收

## Command Model

```bash
watchpid watch <pid> --name train-exp42 --detach
watchpid run --name train-exp42 --detach -- python train.py --config xxx.yaml
watchpid status <watch_id>
watchpid list
watchpid cancel <watch_id>
watchpid notify test
watchpid notify setup
watchpid notify setup --token <pushplus_token>
```

Behavior:

- `watch <pid>`: watch an already-running process. It can reliably tell whether the original process object has exited. It usually cannot provide a trustworthy exit code.
- `run -- <cmd>`: launch the command through `watchpid`. This gives the best metadata quality, including PID, start time, and exit code.
- `cancel <watch_id>`: cancel the watcher, not the target process.
- `--json`: return machine-readable output for agents and scripts.

语义说明：

- `watch <pid>`：监听已经在运行的进程。它能比较可靠地判断“原来的那个进程实例是否结束了”，但一般拿不到可靠退出码。
- `run -- <cmd>`：由 `watchpid` 启动命令，通知质量最高，能拿到 PID、开始时间和退出码。
- `cancel <watch_id>`：当前定义为取消监听，不主动杀目标进程。
- `--json`：给 agent、脚本、skill、MCP 做机器可读输出。

## State Layout

Default state home:

```text
~/.watchpid/
  config.env
  watches/*.json
  events.jsonl
  logs/*.log
```

You can override the state root with:

```bash
WATCHPID_HOME=/path/to/state
```

This is useful in sandboxes, CI, containers, and multi-instance setups.

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

## Notification Configuration

Priority:

1. environment variables
2. `config.env` in the watch home

Currently supported variables:

```bash
WATCHPID_NOTIFY_CHANNELS=pushplus
WATCHPID_PUSHPLUS_TOKEN=xxxx
```

Setup flows:

```bash
watchpid notify setup
watchpid notify setup --token <pushplus_token>
watchpid notify test
```

When no token is configured, `watchpid` prints a clear hint instead of failing silently.

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

## Build

```bash
go build ./...
go build -o watchpid ./cmd/watchpid
```

Cross-compile example:

```bash
GOOS=windows GOARCH=amd64 go build ./...
```

## Agent Integration

Recommended behavior for agents:

1. If the task has not started yet, prefer:

```bash
watchpid run --detach --json -- <command> [args...]
```

2. If the task is already running and the user gives you a PID, prefer:

```bash
watchpid watch <pid> --detach --json
```

3. After registration succeeds:

- tell the user the watcher was registered
- keep the returned `watch_id`
- stop polling
- only query later with `watchpid status <watch_id> --json`

对 agent 的建议调用方式：

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

See also:

- [docs/agent-integration.md](/userhome/lhf/Codes/3rdparty/tools/watchpid/docs/agent-integration.md)

## Architecture

Current package layout:

```text
cmd/watchpid/          CLI entrypoint
internal/cli/          command parsing and presentation
internal/watch/        watch lifecycle and orchestration
internal/store/        persistent watch state and events
internal/process/      OS-specific process inspection
internal/notify/       notifier abstraction and PushPlus
internal/daemon/       detached watcher launch per platform
internal/model/        shared data structures
```

目录分层：

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

## Next Steps

Planned follow-ups:

- verify detached runtime behavior on a normal host outside this sandbox
- improve notification body formatting
- add richer duration / log-tail metadata
- add more channels after PushPlus
- add tests around state transitions and config parsing

后续优先项：

- 在正常主机上验证后台 detach 运行时行为
- 优化通知内容格式
- 增加运行时长和日志尾部摘要
- 在 PushPlus 之后补充更多消息渠道
- 给状态流转和配置解析补测试
