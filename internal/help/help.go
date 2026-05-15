package help

import "fmt"

func Print(version string) {
	fmt.Printf(`fprint-menu %s

A terminal UI for Linux fingerprint management with fprintd.

Usage:
  fprint-menu
  fprint-menu --lang ru
  fprint-menu --lang zh
  fprint-menu --help
  fprint-menu --version

Keys:
  [j]/[down]  move down
  [k]/[up]    move up
  [l]/[right] select
  [h]/[left]  back
  [enter]     select/confirm
  [esc]       back/cancel
  [q]         quit

Russian layout aliases:
  [о]/[л]/[д]/[р] mirror [j]/[k]/[l]/[h] physical positions, [й] mirrors [q]

`, version)
}
