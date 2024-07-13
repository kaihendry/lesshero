package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func getTimes(commits []LHcommit) (times []string) {
	for i := 0; i < len(commits); i++ {
		// if time is unset warn
		if commits[i].Date.IsZero() {
			slog.Warn("commit", "time", "is zero", "hash", commits[i].ShortHash, "author", commits[i].Author)
		}
		times = append(times, commits[i].Date.Format("2006-01-02"))
	}

	// log level debug prints the commit time
	// print first time
	slog.Debug("first", "time", times[0])
	// print last time
	slog.Debug("last", "time", times[len(times)-1])
	return times
}

func getSlocs(commits []LHcommit) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < len(commits); i++ {
		items = append(items, opts.LineData{Value: commits[i].runningTotal, Name: commits[i].ShortHash})
	}
	// print first item
	slog.Debug("first", "item", items[0].Value)
	// print last item
	slog.Debug("last", "item", items[len(items)-1].Value)
	return items
}

func getDecSlocs(commits []LHcommit) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < len(commits); i++ {
		if commits[i].Net < 0 || i < len(commits)-1 && commits[i+1].Net < 0 {
			items = append(items, opts.LineData{Value: commits[i].runningTotal, Name: commits[i].ShortHash})
		} else {
			items = append(items, opts.LineData{Value: "-", Name: commits[i].ShortHash})
		}
	}
	return items
}

func chartHero(commits []LHcommit, gitSrc, fn string, total int) error {
	slog.Debug("commits", "count", len(commits))
	slog.Debug("first", "date", commits[0].Date.Format("2006-01-02"), "hash", commits[0].ShortHash, "total", commits[0].Net)
	slog.Debug("last", "date", commits[len(commits)-1].Date.Format("2006-01-02"), "hash", commits[len(commits)-1].ShortHash, "total", commits[len(commits)-1].Net)

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{PageTitle: "Less Hero"}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type: "inside",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			TriggerOn: "mousemove|click",
			Show:      opts.Bool(true),
			Formatter: "{b}: {c}",
		}),
		charts.WithTitleOpts(opts.Title{Title: gitSrc, Link: "https://github.com/kaihendry/lesshero"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(false)}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Code Count: " + strconv.Itoa(total),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			Data: getTimes(commits),
			AxisLabel: &opts.AxisLabel{
				Rotate: 20,
				Show:   opts.Bool(true),
			},
		}),
	)

	line.SetXAxis(getTimes(commits)).AddSeries("SLOC", getSlocs(commits), charts.WithLineStyleOpts(opts.LineStyle{Color: "red"})).AddSeries("SLOC", getDecSlocs(commits), charts.WithLineStyleOpts(opts.LineStyle{Color: "green"}))

	dynamicFn := fmt.Sprintf(
		`goecharts_%s.on('click', function (params) { navigator.clipboard.writeText(params.name); console.log(params.name, "copied to clipboard"); });`,
		line.ChartID,
	)
	line.AddJSFuncs(dynamicFn)

	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	err = line.Render(f)
	if err != nil {
		return err
	}
	return f.Close()
}
