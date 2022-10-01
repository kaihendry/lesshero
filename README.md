# Less Hero!

Demo: https://s.natalian.org/2022-09-25/chart.html of a checkout of https://git.suckless.org/dwm/

## Goal: Highlight efforts to reduce bloat

It's important for your health to **watch your weight**.

Same for code.

Aim of this project is to celebrate those who refactor and put code on
a diet!

## Install

Assuming you a Go runtime installed:

    go install github.com/kaihendry/lesshero@latest
    lesshero -c chart.html /path/to/a/full/git/checkout

### Docker

    docker run -e FORCE_COLOR=1 -v $(pwd):/repo hendry/lesshero -c /repo/chart.html /repo

## Meta

- https://kaihendry.github.io/lesshero/dev/bench/
