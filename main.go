package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

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

	highlightHero(commits)

	if chartPath != "" {
		// calculate running total, starting from the end (beginning) of commits
		runningTotal := 0
		for i := 0; i < len(commits); i++ {
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

	// log level debug prints the commit time
	if os.Getenv("LOG_LEVEL") == "debug" {
		// print first time
		fmt.Printf("first %s\n", times[0])
		// print last time
		fmt.Printf("last %s\n", times[len(times)-1])
	}
	return times
}

func getSlocs(commits []Commit) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := 0; i < len(commits); i++ {
		items = append(items, opts.LineData{Value: commits[i].runningTotal, Name: commits[i].hash})
	}
	if os.Getenv("LOG_LEVEL") == "debug" {
		// print first item
		fmt.Printf("first %d\n", items[0].Value)
		// print last item
		fmt.Printf("last %d\n", items[len(items)-1].Value)
	}
	return items
}

func chartHero(commits []Commit, gitSrc, fn string) error {

	if os.Getenv("LOG_LEVEL") == "debug" {
		// number of commits to show
		fmt.Printf("Number of commits %d\n", len(commits))
		// print first commit
		fmt.Printf("%s %s %d\n", commits[0].date.Format("2006-01-02"), commits[0].hash, commits[0].total)
		// print last commit
		fmt.Printf("%s %s %d\n", commits[len(commits)-1].date.Format("2006-01-02"), commits[len(commits)-1].hash, commits[len(commits)-1].total)
	}

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
		`goecharts_%s.on('click', function (params) { navigator.clipboard.writeText(params.name); console.log(params.name, "copied to clipboard"); });`,
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

	if os.Getenv("LOG_LEVEL") == "debug" {
		log.Printf("Totalling %d commits", count)
	}

	semaphore := make(chan bool, 1)
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
			if os.Getenv("LOG_LEVEL") == "debug" {
				log.Printf("commit time: %v total: %d", commits[countIndex].date.Format("2006-01-02"), total)
			}
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
