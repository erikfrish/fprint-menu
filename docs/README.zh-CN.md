# FPrint Control

一个基于 `fprintd` 的 Linux 指纹管理终端界面。

Translations: [English](../README.md), [Русский](README.ru.md), [简体中文](README.zh-CN.md)

FPrint Control 适合从桌面启动器或终端中使用。它可以查看已登记指纹、登记新指纹、验证传感器、删除单个槽位、清空本地用户、重启 `fprintd`，并在菜单内运行基础诊断。

## 截图

![主菜单](../screenshots/menu.png)

![Sudo 认证](../screenshots/sudo_window.png)

![指纹登记](../screenshots/fingerprint_scanning.png)

## 功能

- 基于 Bubble Tea 的键盘优先界面。
- 彩色的概览、确认、运行中和结果界面。
- 双栏布局，侧边栏持续显示状态。
- 最小终端尺寸保护；窗口过小时显示调整窗口大小的提示，而不是渲染损坏的边框。
- 鼠标或触控板的垂直滚动可移动选择；点击和水平滚动会被忽略。
- 侧边栏显示软件包状态和已登记指纹。
- 验证和删除流程只显示已经登记的指纹。
- 对需要权限的操作提供内置 sudo/PAM 认证弹窗。
- 登记指纹时使用仅密码认证，避免在扫描前污染指纹读取器状态。
- 登记进度、重试反馈、重复指纹检测和成功界面。
- 通过 SQLite seeds 提供英文、俄文和简体中文本地化，可由 `LANG` 或 `--lang` 选择。
- 支持英文和俄文键盘布局导航。
- 可使用检测到的包管理器安装缺失的 `fprintd`/`libfprint` 组件。
- 危险操作需要确认，并能从 `fprintd` 设备占用状态恢复。
- 诊断软件包、`fprintd.service` 和可能的 USB 指纹设备。

## 安装

```bash
go install github.com/erikfrish/fprint-menu@latest
```

本地构建：

```bash
go build -o fprint-menu .
./fprint-menu
```

## 要求

- `fprintd`
- `libfprint`
- `pacman`、`dnf`、`apt-get` 或 `nix-env` 之一，用于依赖检测和安装提示。
- 可选：`usbutils`，用于更详细的 USB 设备诊断。

请以普通桌面用户运行。不要用 `sudo` 启动整个 TUI；应用只会在具体操作需要权限时请求提权。

## 用法

```bash
fprint-menu
fprint-menu --lang ru
fprint-menu --lang zh
fprint-menu --log debug
fprint-menu --help
```

导航支持方向键、`hjkl` 和俄文键盘布局中的对应物理按键。`q` 在非输入弹窗中退出；`esc` 返回或取消当前弹窗/操作。

登记指纹时会刻意要求密码，而不是使用 sudo 指纹认证。这样可以避免在 `fprintd-enroll` 前立即占用指纹读取器，否则读取器可能保持旧状态，导致登记从错误阶段开始。

日志默认关闭。使用 `--log debug` 可将 debug traces 写入 `/tmp/fprint-menu.log`，用于诊断 sudo/PAM 或 `fprintd` 行为。

## 兼容性

测试日期：2026-05-15。

- Host Arch Linux：通过 `gofmt`、`go test ./...`、`go build ./...`、`go vet ./...`、`go install`、`--help`、`--version`、`--lang ru`、`--lang en`。
- 硬件：ThinkPad T14p Gen 3。
- Fedora Toolbox distrobox：安装 `golang`、`git`、`fprintd`、`fprintd-pam`、`libfprint`；通过 `go build`、`go vet`、`--help`、`--version`、`--lang ru`。
- Ubuntu 24.04 distrobox：安装 `golang-go`、`git`、`fprintd`、`libpam-fprintd`、`libfprint-2-2`；通过 `go build`、`go vet`、`--help`、`--version`、`--lang ru`。
- Nix container：`nixos/nix` 因缺少 `/etc/os-release` 不兼容 distrobox，因此直接通过 `podman` 和 `nix-shell -p go git` 测试；通过 `go build`、`go vet`、`--help`、`--version`、`--lang ru`。

运行时包映射：

- Arch：`fprintd`、`libfprint`
- Fedora：`fprintd`、`fprintd-pam`、`libfprint`
- Ubuntu：`fprintd`、`libpam-fprintd`、`libfprint-2-2`
- Nix：`nixpkgs.fprintd`、`nixpkgs.libfprint`
