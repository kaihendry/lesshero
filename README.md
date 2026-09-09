# Less Hero!

[![Less Hero explainer](http://img.youtube.com/vi/Zlsq9B6KdB0/0.jpg)](http://www.youtube.com/watch?v=Zlsq9B6KdB0 "Highlighting less code")

Demo: https://kaihendry.github.io/lesshero/

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

Exclude data files from the entire history:

    lesshero -ignore all.json,static/all.json /path/to/a/full/git/checkout

Paths are exact, relative to the repository root, and separated by commas.
Include previous names when a file has moved. The same exclusions apply to every
commit, including additions, deletions, and renames. No files or Git history are
changed. When reading existing JSONL, counts have already been computed; apply
`-ignore` when generating the JSONL from the repository.

### Docker

    docker run -v $(pwd):/repo hendry/lesshero -o /repo/chart.html /repo

## Related projects to help track code complexity

- https://github.com/boyter/scc
- https://github.com/kaihendry/graphsloc

## Create a SLOC chart via a GitHub action

First you need to enable Github Pages with the source of **Github Actions**.

The action accepts the same exclusions through its `ignore` input:

```yaml
- uses: kaihendry/lesshero@main
  with:
    ignore: all.json,static/all.json
```
