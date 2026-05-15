# FPrint Control

TUI для управления отпечатками в Linux через `fprintd`.

Переводы: [English](../README.md), [Русский](README.ru.md), [简体中文](README.zh-CN.md)

Утилита рассчитана на запуск из терминала или desktop launcher: показывает зарегистрированные отпечатки в sidebar, добавляет или перезаписывает отпечаток, проверяет сенсор, удаляет одну запись, очищает все отпечатки и запускает базовую диагностику.

## Возможности

- Keyboard-first интерфейс на Bubble Tea.
- Двухпанельный layout с постоянным sidebar статуса.
- Минимальный размер окна защищает от сломанной рамки и показывает предупреждение при слишком маленьком терминале.
- Вертикальный scroll мыши или touchpad двигает курсор; клики и горизонтальный scroll игнорируются.
- Sidebar показывает состояние пакетов и зарегистрированные отпечатки.
- Проверка и удаление показывают только уже зарегистрированные отпечатки.
- Действия, меняющие систему, сначала запрашивают подтверждение перед `sudo`.
- Локализация EN/RU/ZH через SQLite seeds, выбор по `LANG` или `--lang`.
- Навигация работает на английской и русской раскладках.
- Установка недостающих `fprintd`/`libfprint` компонентов через найденный пакетный менеджер.
- Диагностика пакетов, `fprintd.service` и вероятных USB-устройств отпечатков.

## Установка

```bash
go install github.com/erikfrish/fprint-menu@latest
```

Локальная сборка:

```bash
go build -o fprint-menu .
./fprint-menu
```

## Требования

- `fprintd`
- `libfprint`
- Один из `pacman`, `dnf`, `apt-get` или `nix-env` для проверки и установки зависимостей.
- Опционально: `usbutils` для более подробной диагностики USB.

Запускай из терминала. Для добавления, удаления, очистки и установки зависимостей приложение само предложит перейти к `sudo`-аутентификации.

## Совместимость

Проверено 2026-05-15:

- Host Arch Linux: `gofmt`, `go build`, `go vet`, `go install`, `--help`, `--version`, `--lang ru`, `--lang en`, `--lang zh`.
- Железо: ThinkPad T14p Gen 3.
- Fedora Toolbox distrobox: установлены `golang`, `git`, `fprintd`, `fprintd-pam`, `libfprint`; прошли `go build`, `go vet`, `--help`, `--version`, `--lang ru`.
- Ubuntu 24.04 distrobox: установлены `golang-go`, `git`, `fprintd`, `libpam-fprintd`, `libfprint-2-2`; прошли `go build`, `go vet`, `--help`, `--version`, `--lang ru`.
- Nix container: `nixos/nix` несовместим с distrobox из-за отсутствия `/etc/os-release`, поэтому проверен напрямую через `podman` и `nix-shell -p go git`; прошли `go build`, `go vet`, `--help`, `--version`, `--lang ru`.

Пакеты runtime:

- Arch: `fprintd`, `libfprint`
- Fedora: `fprintd`, `fprintd-pam`, `libfprint`
- Ubuntu: `fprintd`, `libpam-fprintd`, `libfprint-2-2`
- Nix: `nixpkgs.fprintd`, `nixpkgs.libfprint`
