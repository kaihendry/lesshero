package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/fluxcd/go-git/v5"
	"github.com/fluxcd/go-git/v5/plumbing/object"

	"github.com/jwalton/gchalk"
)

var (
	version  = "dev"
	commit   = "none"
	repoPath = "." // pwd is default
)

type Commit struct {
	hash         string
	author       string
	date         time.Time
	total        int
	runningTotal int
}

func main() {
	var chartPath string
	flag.StringVar(&chartPath, "c", "", "path to html chart output")
	highlight := flag.Bool("hl", true, "highlight")
	flag.Usage = func() {
		if commit == "none" {
			fmt.Fprintf(os.Stderr, "go version -m ~/go/bin/lesshero\n")
		} else {
			fmt.Fprintf(os.Stderr, "https://github.com/kaihendry/lesshero %s (%s)\n", version, commit)
		}
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.Arg(0) != "" {
		repoPath = flag.Arg(0)
	}

	commits, gitSrc, err := lessHero(repoPath)
	if err != nil {
		log.Fatalf("%s: %v", repoPath, err)
	}

	if *highlight {
		highlightHero(commits)
	}

	if chartPath != "" {
		// calculate running total, starting from the end (beginning) of commits
		runningTotal := 0
		for i := len(commits) - 1; i >= 0; i-- {
			runningTotal += commits[i].total
			commits[i].runningTotal = runningTotal
		}
		err = chartHero(commits, gitSrc, chartPath)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getTimes(commits []Commit) (times []string) {
	for i := 0; i < len(commits); i++ {
		times = append(times, commits[i].date.Format("2006-01-02"))
	}
	return times
}

func getSlocs(commits []Commit) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := len(commits) - 1; i >= 0; i-- {
		items = append(items, opts.LineData{Value: commits[i].runningTotal, Name: commits[i].hash})
	}
	return items
}

func chartHero(commits []Commit, gitSrc, fn string) error {
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
			Formatter: "{b}",
		}),
		charts.WithTitleOpts(opts.Title{Title: gitSrc}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Code Count",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Rotate: 20,
			},
		}),
	)
	line.SetXAxis(getTimes(commits)).AddSeries("SLOC", getSlocs(commits))

	dynamicFn := fmt.Sprintf(
		`goecharts_%s.on('click', function (params) {   navigator.clipboard.writeText(params.name); console.log(params.name, "copied to clipboard"); });`,
		line.ChartID,
	)
	line.AddJSFuncs(dynamicFn)

	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	line.Render(f)
	return nil
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

	log.Printf("Totalling %d commits", count)

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
				// warn and ignore error
				log.Printf("c.Stats: %v, index: %d, c: %v", err, countIndex, c.Hash.String()[:7])
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
			// log.Printf("commit time: %v", commits[countIndex].date.Format("2006-01-02 15:04:05"))
		}(c, countIndex)
		countIndex++
		return nil
	})

	wg.Wait()

	// sort commits by ascending date
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
