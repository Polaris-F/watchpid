# Agent Integration Draft

`watchpid` 的目标不是替代 agent，而是把“长任务状态盯梢”从 agent 身上移走。

## 给 agent 的调用规则

1. 如果任务还没启动，优先用：

```bash
watchpid run --detach --json -- <command> [args...]
```

2. 如果任务已经在跑，并且用户给了 PID，优先用：

```bash
watchpid watch <pid> --detach --json
```

3. 注册成功后，agent 应该：

- 告诉用户监听已经注册成功
- 记录 `watch_id`
- 不继续轮询这个进程
- 只在用户回头询问时，用 `watchpid status <watch_id> --json` 查询

4. 如果没有配置通知 token，agent 应该优先提示：

```bash
watchpid notify setup --token <pushplus_token> --json
```

或者设置环境变量：

```bash
export WATCHPID_PUSHPLUS_TOKEN=<pushplus_token>
```

## 给 skill / MCP 的行为约束

- 不要在注册成功后继续 `wait`
- 不要把“启动任务”和“持续轮询任务”绑在同一个 agent 生命周期里
- 优先消费 `--json` 输出
- 默认把 `cancel` 理解为“取消监听”，不是“杀进程”

## 建议暴露给上层的最小命令集

- `watchpid run --detach --json -- <cmd>`
- `watchpid watch <pid> --detach --json`
- `watchpid status <watch_id> --json`
- `watchpid list --json`
- `watchpid cancel <watch_id> --json`
- `watchpid notify test --json`
