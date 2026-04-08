[中文](./README.md) | [English](./README_EN.md)

# watchpid

`watchpid` is a lightweight watcher for long-running jobs. Its job is not to replace an agent, but to take over the boring part: watching whether a process has finished. Once a watch is registered, the agent can stop waiting and let `watchpid` persist state and send notifications.

## Why

Typical agent behavior for long tasks today looks like this:

- start a long-running command
- check that it seems healthy
- keep waiting, polling, or saying “I’ll tell you when it finishes”

That is the wrong ownership boundary.

`watchpid` is meant to do this instead:

- use `watchpid watch <pid>` for a process that is already running
- use `watchpid run -- <cmd>` for a command that should be launched now
- persist watcher state locally
- notify when the target finishes

## Current Status

This repository is the first usable skeleton.

Current scope:

- Go CLI
- Linux and Windows build targets
- persistent local watch state
- PushPlus as the first notification channel
- JSON output for agent / skill / MCP integration

What is already in place:

- CLI skeleton
- clear layering for `store / config / process / notify`
- Linux and Windows builds compile successfully
- foreground flows have been verified locally

Known limitation in this environment:

- detached background validation is limited by the current sandbox, which reaps child processes aggressively

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

- `watch <pid>`: watch an already-running process. It can reliably tell whether the original process object has exited, but it usually cannot provide a trustworthy exit code.
- `run -- <cmd>`: launch the command through `watchpid`. This gives the best metadata quality, including PID, start time, and exit code.
- `cancel <watch_id>`: cancel the watcher, not the target process.
- `--json`: return machine-readable output for agents and scripts.

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

## Notification Configuration

Priority:

1. environment variables
2. `config.env` in the watch home

Currently supported:

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

If no token is configured, `watchpid` prints a clear hint instead of failing silently.

## Build

```bash
go build ./...
go build -o watchpid ./cmd/watchpid
```

Windows cross-compile example:

```bash
GOOS=windows GOARCH=amd64 go build ./...
```

## Recommended Agent Flow

1. If the task has not started yet, prefer:

```bash
watchpid run --detach --json -- <command> [args...]
```

2. If the task is already running and the user gives you a PID, prefer:

```bash
watchpid watch <pid> --detach --json
```

3. After registration succeeds, the agent should:

- tell the user that the watcher was registered
- keep the returned `watch_id`
- stop polling
- query later with `watchpid status <watch_id> --json` only when needed

See also:

- [docs/agent-integration.md](./docs/agent-integration.md)

## Project Layout

```text
cmd/watchpid/          CLI entrypoint
internal/cli/          command parsing and output
internal/watch/        watch lifecycle and orchestration
internal/store/        persistent state and events
internal/process/      OS-specific process inspection
internal/notify/       notifier abstraction and PushPlus
internal/daemon/       detached watcher launch per platform
internal/model/        shared models
```

## Next Steps

- verify detached runtime behavior on a normal host outside this sandbox
- improve notification formatting
- add richer duration and log-tail metadata
- add more notification channels after PushPlus
- add tests for state transitions and config parsing
