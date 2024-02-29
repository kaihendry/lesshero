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
    lesshero -o chart.html /path/to/a/full/git/checkout

For more options:
```
lesshero -h
```

### Docker

    docker run -e FORCE_COLOR=1 -v $(pwd):/repo hendry/lesshero -o /repo/chart.html /repo

## Related projects to help track code complexity

- https://github.com/boyter/scc
- https://github.com/kaihendry/graphsloc

## Create a SLOC chart via a GitHub action

```yaml
name: Create a Less Hero SLOC chart

on: [ workflow_dispatch, push ]

# Sets permissions of the GITHUB_TOKEN to allow deployment to GitHub Pages
permissions:
  contents: read
  pages: write
  id-token: write

# Allow only one concurrent deployment, skipping runs queued between the run in-progress and latest queued.
# However, do NOT cancel in-progress runs as we want to allow these production deployments to complete.
concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  # Single deploy job since we're just deploying
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Setup Pages
        uses: actions/configure-pages@v1
      - uses: kaihendry/lesshero@v1
      - run:  mkdir "_site" && mv lesshero.html _site/index.html
      - name: Upload artifact # TODO: we want to ideally upload one file, not overwrite the whole site!
        uses: actions/upload-pages-artifact@v3
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```
