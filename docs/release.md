# 发布与安装

`watchpid` 现在已经具备最小可发布形态：

- 可以通过 `go install` 直接安装
- 可以本地打包 Linux / Windows 发布产物
- 可以在 GitHub tag 发布时自动上传 Release 附件

## 本地构建

```bash
make test
make build
./bin/watchpid version
```

## 直接安装

如果目标机器已经安装 Go：

```bash
go install github.com/Polaris-F/watchpid/cmd/watchpid@latest
watchpid version
```

如果目标机器不希望安装 Go：

1. 到 GitHub Releases 下载对应平台包
2. Linux 解压 `watchpid_<version>_linux_<arch>.tar.gz`
3. Windows 解压 `watchpid_<version>_windows_<arch>.zip`
4. 把二进制加入 `PATH`

Linux 示例：

```bash
tar -xzf watchpid_v0.1.0_linux_amd64.tar.gz
install -m 0755 watchpid_v0.1.0_linux_amd64/watchpid ~/.local/bin/watchpid
watchpid version
```

## 首次配置

安装后，至少建议完成通知配置：

```bash
watchpid notify setup --token <pushplus_token>
watchpid notify test
```

也可以直接使用环境变量：

```bash
export WATCHPID_PUSHPLUS_TOKEN=<pushplus_token>
export WATCHPID_NOTIFY_CHANNELS=pushplus
```

## 本地打包发布

生成发布产物：

```bash
make release VERSION=v0.1.0
```

脚本默认尝试构建这些目标：

```text
linux/amd64
linux/arm64
windows/amd64
windows/arm64
```

如果当前 Go toolchain 不支持某个目标，脚本会自动跳过。

产物会写到：

```text
dist/
  watchpid_v0.1.0_linux_amd64.tar.gz
  watchpid_v0.1.0_linux_arm64.tar.gz
  watchpid_v0.1.0_windows_amd64.zip
  sha256sums.txt
```

## GitHub Release 发布

仓库已经可以配合 GitHub Actions 使用 tag 发布：

```bash
git tag v0.1.0
git push origin v0.1.0
```

推送 tag 后，工作流会：

- 运行测试
- 构建当前工具链支持的 Linux / Windows 发布包
- 上传到对应 GitHub Release

## 当前仍需注意

- 当前沙箱环境无法完整验证 `--detach` 后台路径，不适合作为后台行为的最终验收环境
- 目前还没有自动化测试覆盖核心状态流转
