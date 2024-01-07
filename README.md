# Less Hero!

[![Less Hero explainer](http://img.youtube.com/vi/Zlsq9B6KdB0/0.jpg)](http://www.youtube.com/watch?v=Zlsq9B6KdB0 "Highlighting less code")

Demo: https://s.natalian.org/2024-01-07/chart.html of a checkout of https://git.suckless.org/dwm/

## Goal: Highlight efforts to reduce bloat

It's important for your health to **watch your weight**.

Same for code.

Aim of this project is to celebrate those who refactor and put code on
a diet!

## Install

Assuming you have a Go runtime installed:

    go install github.com/kaihendry/lesshero@latest
    lesshero -c chart.html /path/to/a/full/git/checkout

For more options:
```
lesshero -h
```

### Docker

    docker run -e FORCE_COLOR=1 -v $(pwd):/repo hendry/lesshero -c /repo/chart.html /repo

## Meta

- https://kaihendry.github.io/lesshero/dev/bench/

## Related projects to help track code complexity

- https://github.com/boyter/scc
- https://github.com/kaihendry/graphsloc
