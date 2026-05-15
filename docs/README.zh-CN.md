# FPrint Control

一个基于 `fprintd` 的 Linux 指纹管理终端界面。

Translations: [English](../README.md), [Русский](README.ru.md), [简体中文](README.zh-CN.md)

它适合从桌面启动器或终端中使用：在侧边栏显示已登记的指纹，添加或覆盖指纹，验证传感器，删除单个记录，清空所有指纹，并运行基础诊断。

## 功能

- 基于 Bubble Tea 的键盘优先界面。
- 双栏布局，右侧始终显示状态侧边栏。
- 最小终端尺寸保护；窗口过小时显示调整窗口大小的提示，而不是渲染损坏的边框。
- 鼠标或触控板的垂直滚动可移动选择；点击和水平滚动会被忽略。
- 侧边栏显示软件包状态和已登记指纹。
- 验证和删除流程只显示已经登记的指纹。
- 会修改系统的操作会先确认，再交给 `sudo` 认证。
- 通过 SQLite seeds 提供英文、俄文和简体中文本地化，可由 `LANG` 或 `--lang` 选择。
- 支持英文和俄文键盘布局导航。
- 可使用检测到的包管理器安装缺失的 `fprintd`/`libfprint` 组件。
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

请从终端运行。添加、删除、清空和安装依赖时，应用会提示是否进入 `sudo` 认证。

## 兼容性

测试日期：2026-05-15。

- Host Arch Linux：通过 `gofmt`、`go build`、`go vet`、`go install`、`--help`、`--version`、`--lang ru`、`--lang en`、`--lang zh`。
- 硬件：ThinkPad T14p Gen 3。
- Fedora Toolbox distrobox：安装 `golang`、`git`、`fprintd`、`fprintd-pam`、`libfprint`；通过 `go build`、`go vet`、`--help`、`--version`、`--lang ru`。
- Ubuntu 24.04 distrobox：安装 `golang-go`、`git`、`fprintd`、`libpam-fprintd`、`libfprint-2-2`；通过 `go build`、`go vet`、`--help`、`--version`、`--lang ru`。
- Nix container：`nixos/nix` 因缺少 `/etc/os-release` 不兼容 distrobox，因此直接通过 `podman` 和 `nix-shell -p go git` 测试；通过 `go build`、`go vet`、`--help`、`--version`、`--lang ru`。

运行时包映射：

- Arch：`fprintd`、`libfprint`
- Fedora：`fprintd`、`fprintd-pam`、`libfprint`
- Ubuntu：`fprintd`、`libpam-fprintd`、`libfprint-2-2`
- Nix：`nixpkgs.fprintd`、`nixpkgs.libfprint`
