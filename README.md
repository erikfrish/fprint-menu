# FPrint Control

A terminal UI for Linux fingerprint management powered by `fprintd`.

Translations: [English](README.md), [Русский](docs/README.ru.md), [简体中文](docs/README.zh-CN.md)

It is designed for desktop launchers and terminal workflows: see enrolled prints
in the sidebar, enroll a finger, verify the sensor, delete one slot, wipe all
prints, and run basic diagnostics without leaving the menu.

## Features

- Keyboard-first Bubble Tea interface
- Colorful overview, confirmation, running, and result screens
- Two-panel layout with a persistent status sidebar
- Minimum terminal size guard shows a resize warning instead of broken layout
- Vertical mouse/touchpad scrolling moves selection; clicks and horizontal scroll are ignored
- Sidebar shows package health and currently enrolled fingerprints
- Verify and delete flows only show fingerprints that are already enrolled
- Mutating actions ask for confirmation before handing off to `sudo` authentication
- SQLite-backed English/Russian/Simplified Chinese localization selected from `LANG` or `--lang`
- Keyboard navigation works on English and Russian layouts
- Returns to the menu after information and command screens
- Installs missing `fprintd`/`libfprint` packages with the detected package manager when requested
- Destructive actions require confirmation
- Diagnostics for packages, `fprintd.service`, and likely USB fingerprint devices

## Install

```bash
go install github.com/erikfrish/fprint-menu@latest
```

Local build:

```bash
go build -o fprint-menu .
./fprint-menu
```

## Requirements

- `fprintd`
- `libfprint`
- One of `pacman`, `dnf`, `apt-get`, or `nix-env` for dependency detection/install prompts
- Optional: `usbutils` for richer device diagnostics

Run from a terminal. For privileged enrollment/deletion, launch through `sudo`
or a desktop entry that wraps the binary with `sudo`.

## Compatibility

Tested on 2026-05-15:

- Host Arch Linux: `gofmt`, `go build`, `go vet`, `go install`, `--help`, `--version`, `--lang ru`, `--lang en`.
- Hardware: ThinkPad T14p Gen 3.
- Fedora Toolbox distrobox: installed `golang`, `git`, `fprintd`, `fprintd-pam`, `libfprint`; `go build`, `go vet`, `--help`, `--version`, `--lang ru` passed.
- Ubuntu 24.04 distrobox: installed `golang-go`, `git`, `fprintd`, `libpam-fprintd`, `libfprint-2-2`; `go build`, `go vet`, `--help`, `--version`, `--lang ru` passed.
- Nix container: `nixos/nix` is not distrobox-compatible because it lacks `/etc/os-release`, so it was tested directly with `podman` and `nix-shell -p go git`; `go build`, `go vet`, `--help`, `--version`, `--lang ru` passed.

Runtime package mapping:

- Arch: `fprintd`, `libfprint`
- Fedora: `fprintd`, `fprintd-pam`, `libfprint`
- Ubuntu: `fprintd`, `libpam-fprintd`, `libfprint-2-2`
- Nix: `nixpkgs.fprintd`, `nixpkgs.libfprint`
