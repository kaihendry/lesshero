package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sort"
    "strconv"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/fluxcd/go-git/v5"
	"github.com/fluxcd/go-git/v5/plumbing/object"

	"github.com/jwalton/gchalk"
    "github.com/pkg/browser"
)

var (
	repoPath = "." // pwd is default
)

func getLogger(logLevel string) *slog.Logger {
	levelVar := slog.LevelVar{}

	if logLevel != "" {
		if err := levelVar.UnmarshalText([]byte(logLevel)); err != nil {
			panic(fmt.Sprintf("Invalid log level %s: %v", logLevel, err))
		}
	}

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: levelVar.Level(),
	}))
}

type Commit struct {
	hash         string
	author       string
	date         time.Time
	total        int
	runningTotal int
}

func main() {
	slog.SetDefault(getLogger(os.Getenv("LOGLEVEL")))

    var chartPath string
    var autoOpenChart bool
    var showCommitsHighlight bool

	flag.StringVar(&chartPath, "c", "chart.html", "path to html chart output")
    flag.BoolVar(&autoOpenChart, "b", false, "auto open chart in default browser")
    flag.BoolVar(&showCommitsHighlight, "l", false, "show list of commits highlighted based on code count change")
	flag.Parse()

	if flag.Arg(0) != "" {
		repoPath = flag.Arg(0)
	}

	commits, gitSrc, err := lessHero(repoPath)
	if err != nil {
		slog.Error("lessHero", "err", err, "repoPath", repoPath)
        fmt.Println(" - Directory", repoPath, "is not a git repository!")
		return
	}

    if showCommitsHighlight {
        highlightHero(commits)
    }

    // calculate running total, starting from the end (beginning) of commits
    runningTotal := 0
    for i := 0; i < len(commits); i++ {
        runningTotal += commits[i].total
        commits[i].runningTotal = runningTotal
    }
    err = chartHero(commits, gitSrc, chartPath, runningTotal)
    if err != nil {
        slog.Error("charthero", "err", err)
        return
    }

    if autoOpenChart {
        err = browser.OpenFile(chartPath)
        if err != nil {
            slog.Error("charthero open", "err", err)
            return
        }
    }
}

func getTimes(commits []Commit) (times []string) {
	for i := 0; i < len(commits); i++ {
		times = append(times, commits[i].date.Format("2006-01-02"))
	}

	// log level debug prints the commit time
	// print first time
	slog.Debug("first", "time", times[0])
	// print last time
	slog.Debug("last", "time", times[len(times)-1])
	return times
}

func getSlocs(commits []Commit) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < len(commits); i++ {
		items = append(items, opts.LineData{Value: commits[i].runningTotal, Name: commits[i].hash})
	}
	// print first item
	slog.Debug("first", "item", items[0].Value)
	// print last item
	slog.Debug("last", "item", items[len(items)-1].Value)
	return items
}

func chartHero(commits []Commit, gitSrc, fn string, total int) error {
	slog.Debug("commits", "count", len(commits))
	slog.Debug("first", "date", commits[0].date.Format("2006-01-02"), "hash", commits[0].hash, "total", commits[0].total)
	slog.Debug("last", "date", commits[len(commits)-1].date.Format("2006-01-02"), "hash", commits[len(commits)-1].hash, "total", commits[len(commits)-1].total)

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{PageTitle: "Less Hero"}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type: "inside",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			TriggerOn: "mousemove|click",
			Show:      true,
            Formatter: "{b}: {c}",
		}),
		charts.WithTitleOpts(opts.Title{Title: gitSrc, Link: "https://github.com/kaihendry/lesshero"}),
		charts.WithLegendOpts(opts.Legend{Show: false}),
		charts.WithYAxisOpts(opts.YAxis{
            Name: "Code Count: " + strconv.Itoa(total),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "category",
			Data: getTimes(commits),
			AxisLabel: &opts.AxisLabel{
				Rotate: 20,
				Show:   true,
                //Color:  "green",
			},
		}),
	)

	line.SetXAxis(getTimes(commits)).AddSeries("SLOC", getSlocs(commits))

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

func lessHero(path string) (commits []Commit, gitSrc string, err error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, "", fmt.Errorf("git.PlainOpen: %w", err)
	}

	remotes, err := r.Remotes()
	if err != nil {
		return nil, "", fmt.Errorf("r.Remotes: %w", err)
	}

	for _, remote := range remotes {
		r := remote.Config()
		if r.Name == "origin" {
			gitSrc = r.URLs[0]
		}
	}

	ref, err := r.Head()
	if err != nil {
		return nil, "", fmt.Errorf("r.Head: %w", err)
	}

	cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, "", fmt.Errorf("r.Log: %w", err)
	}

	count := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})

	if err != nil {
		return nil, "", fmt.Errorf("cIter.ForEach: %w, count: %d", err, count)
	}

	cIter, err = r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, "", fmt.Errorf("r.Log: %w", err)
	}

	commits = make([]Commit, count)

	if os.Getenv("LOG_LEVEL") == "debug" {
		slog.Debug("total", "commits", count)
	}

	// number of cores
	semaphore := make(chan bool, runtime.NumCPU())
	wg := sync.WaitGroup{}
	countIndex := 0

	err = cIter.ForEach(func(c *object.Commit) error {
		wg.Add(1)
		semaphore <- true
		go func(c *object.Commit, countIndex int) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			fStats, err := c.Stats()
			total := 0
			if err != nil {
				slog.Warn("upload failed", "err", err, "index", countIndex, "commit", c.Hash.String()[:7])
			}
			for _, fStat := range fStats {
				total += fStat.Addition - fStat.Deletion
			}
			commits[countIndex] = Commit{
				hash:   c.Hash.String()[:7],
				author: c.Author.Name,
				total:  total,
				date:   c.Author.When,
			}
			slog.Debug("commit", "index", countIndex, "commit", c.Hash.String()[:7], "total", total)
		}(c, countIndex)
		countIndex++
		return nil
	})

	wg.Wait()

	// sort in ascending order
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].date.Before(commits[j].date)
	})

	return
}

func highlightHero(commits []Commit) {
	for _, commit := range commits {
		commitId := fmt.Sprintf(
			"%s %s %s %d %d",
			commit.date.Format("2006-01-02"),
			commit.hash,
			commit.author,
			commit.total,
			commit.runningTotal,
		)
		switch {
		case commit.total < -10:
			fmt.Println(gchalk.WithGreen().Bold(commitId))
		case commit.total < 0:
			fmt.Println(gchalk.Green(commitId))
		case commit.total > 50:
			fmt.Println(gchalk.WithRed().Bold(commitId))
		case commit.total > 10:
			fmt.Println(gchalk.BrightYellow(commitId))
		case commit.total > 0:
			fmt.Println(gchalk.Yellow(commitId))
		}
	}
}
