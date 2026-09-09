package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"os"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// chartHero renders commits in chronological order.
func chartHero(commits []LHcommit, gitSrc, outputFile string) error {
	if len(commits) == 0 {
		return errors.New("no commits to chart")
	}
	dates := make([]string, len(commits))
	slocs := make([]opts.LineData, len(commits))
	decreases := make([]opts.LineData, len(commits))
	tooltips := make([]string, len(commits))
	total := 0
	for i, commit := range commits {
		total += commit.Net
		dates[i] = commit.Date.Format("2006-01-02")
		slocs[i] = opts.LineData{Value: total, Name: commit.ShortHash}
		decreases[i] = slocs[i]
		if commit.Net >= 0 && (i == len(commits)-1 || commits[i+1].Net >= 0) {
			decreases[i].Value = "-"
		}
		tooltips[i] = fmt.Sprintf("%s · %s<br/>%s<br/>%d SLOC (%+d)",
			dates[i], html.EscapeString(commit.ShortHash),
			html.EscapeString(commit.Author), total, commit.Net)
	}
	line := charts.NewLine()
	tooltipJSON, err := json.Marshal(tooltips)
	if err != nil {
		return err
	}
	line.AddJSFuncs(fmt.Sprintf("const tooltips_%s = %s;", line.ChartID, tooltipJSON))
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{PageTitle: "Less Hero"}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type: "inside",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			TriggerOn: "mousemove|click",
			Show:      opts.Bool(true),
			Formatter: opts.FuncOpts(fmt.Sprintf("function (params) { return tooltips_%s[params[0].dataIndex]; }", line.ChartID)),
		}),
		charts.WithTitleOpts(opts.Title{
			Title: gitSrc, Subtitle: "Made with Less Hero",
			SubLink: "https://github.com/kaihendry/lesshero",
		}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(false)}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			AxisLabel: &opts.AxisLabel{
				Rotate: 20,
				Show:   opts.Bool(true),
			},
		}),
	)

	line.SetXAxis(dates).
		AddSeries("SLOC", slocs, charts.WithLineStyleOpts(opts.LineStyle{Color: "red"})).
		AddSeries("SLOC", decreases, charts.WithLineStyleOpts(opts.LineStyle{Color: "green"}))

	dynamicFn := fmt.Sprintf(
		`goecharts_%s.on('click', function (params) { navigator.clipboard.writeText(params.name); console.log(params.name, "copied to clipboard"); });`,
		line.ChartID,
	)
	line.AddJSFuncs(dynamicFn)

	if err := os.WriteFile(outputFile, line.RenderContent(), 0644); err != nil {
		return err
	}
	slog.Info("generated chart", "output", outputFile, "commits", len(commits), "name", gitSrc)
	return nil
}
