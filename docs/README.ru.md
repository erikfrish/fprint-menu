# FPrint Control

TUI для управления отпечатками в Linux через `fprintd`.

Переводы: [English](../README.md), [Русский](README.ru.md), [简体中文](README.zh-CN.md)

FPrint Control рассчитан на запуск из терминала или desktop launcher. Он показывает зарегистрированные отпечатки, добавляет новый отпечаток, проверяет сенсор, удаляет один слот, очищает локальных пользователей, перезапускает `fprintd` и запускает базовую диагностику без выхода из меню.

## Скриншоты

![Главное меню](../screenshots/menu.png)

![Sudo-аутентификация](../screenshots/sudo_window.png)

![Регистрация отпечатка](../screenshots/fingerprint_scanning.png)

## Возможности

- Keyboard-first интерфейс на Bubble Tea.
- Цветные экраны обзора, подтверждения, выполнения и результата.
- Двухпанельный layout с постоянным sidebar статуса.
- Минимальный размер окна защищает от сломанной рамки и показывает предупреждение при слишком маленьком терминале.
- Вертикальный scroll мыши или touchpad двигает курсор; клики и горизонтальный scroll игнорируются.
- Sidebar показывает состояние пакетов и зарегистрированные отпечатки.
- Проверка и удаление показывают только уже зарегистрированные отпечатки.
- Встроенное окно sudo/PAM-аутентификации для привилегированных действий.
- Для регистрации отпечатка используется password-only auth, чтобы сенсор оставался чистым для сканирования.
- Прогресс регистрации, retry feedback, duplicate detection и success screen.
- Локализация EN/RU/ZH через SQLite seeds, выбор по `LANG` или `--lang`.
- Навигация работает на английской и русской раскладках.
- Установка недостающих `fprintd`/`libfprint` компонентов через найденный пакетный менеджер.
- Опасные действия требуют подтверждения и умеют восстанавливаться после занятого `fprintd` device claim.
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

Запускай приложение от обычного desktop user. Не запускай весь TUI через `sudo`: приложение само запросит права только тогда, когда действие действительно требует привилегий.

## Использование

```bash
fprint-menu
fprint-menu --lang ru
fprint-menu --lang zh
fprint-menu --log debug
fprint-menu --help
```

Навигация работает через стрелки, `hjkl` и физически соответствующие клавиши русской раскладки. `q` выходит вне modal inputs; `esc` возвращает назад или отменяет текущее modal/action.

Регистрация отпечатка намеренно просит пароль, а не sudo-аутентификацию отпечатком. Это не даёт sudo использовать сенсор прямо перед `fprintd-enroll`, иначе сенсор может остаться в устаревшем состоянии и регистрация начнётся не с той стадии.

Логирование выключено по умолчанию. Используй `--log debug`, чтобы писать debug traces в `/tmp/fprint-menu.log` при диагностике sudo/PAM или `fprintd`.

## Совместимость

Проверено 2026-05-15:

- Host Arch Linux: `gofmt`, `go test ./...`, `go build ./...`, `go vet ./...`, `go install`, `--help`, `--version`, `--lang ru`, `--lang en`.
- Железо: ThinkPad T14p Gen 3.
- Fedora Toolbox distrobox: установлены `golang`, `git`, `fprintd`, `fprintd-pam`, `libfprint`; прошли `go build`, `go vet`, `--help`, `--version`, `--lang ru`.
- Ubuntu 24.04 distrobox: установлены `golang-go`, `git`, `fprintd`, `libpam-fprintd`, `libfprint-2-2`; прошли `go build`, `go vet`, `--help`, `--version`, `--lang ru`.
- Nix container: `nixos/nix` несовместим с distrobox из-за отсутствия `/etc/os-release`, поэтому проверен напрямую через `podman` и `nix-shell -p go git`; прошли `go build`, `go vet`, `--help`, `--version`, `--lang ru`.

Пакеты runtime:

- Arch: `fprintd`, `libfprint`
- Fedora: `fprintd`, `fprintd-pam`, `libfprint`
- Ubuntu: `fprintd`, `libpam-fprintd`, `libfprint-2-2`
- Nix: `nixpkgs.fprintd`, `nixpkgs.libfprint`
