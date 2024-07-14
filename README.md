# Less Hero!

[![Less Hero explainer](http://img.youtube.com/vi/Zlsq9B6KdB0/0.jpg)](http://www.youtube.com/watch?v=Zlsq9B6KdB0 "Highlighting less code")

Demo: https://kaihendry.github.io/scc/

## Goal: Highlight efforts to reduce bloat

It's important for your health to **watch your weight**.

Same for code.

Aim of this project is to celebrate those who refactor and put code on
a diet!

## Install

Assuming you have a Go runtime installed:

    go install github.com/kaihendry/lesshero@latest

## Usage

`lesshero -b` will show source code additions over time in a chart in your browser.

Explicit usage:

    lesshero /path/to/a/full/git/checkout > sloc.jsonl
    cat sloc.jsonl | lesshero -o chart.html

### Docker

    docker run -v $(pwd):/repo hendry/lesshero -o /repo/chart.html /repo

## Related projects to help track code complexity

- https://github.com/boyter/scc
- https://github.com/kaihendry/graphsloc

## Create a SLOC chart via a GitHub action

First you need to enable Github Pages with the source of **Github Actions**.