package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jwalton/gchalk"
)

type Commit struct {
	Hash         string
	Author       string
	Date         time.Time
	Total        int
	RunningTotal int
}

func main() {
	var chartPath string
	repoPath := "."
	flag.StringVar(&chartPath, "c", "", "path to html chart output")
	flag.Parse()

	if flag.Arg(0) != "" {
		repoPath = flag.Arg(0)
	}

	// check path exists else exit
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		log.Fatal(err)
	}

	commits, err := lessHero(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	// calculate running total, starting from the end (beginning) of commits
	runningTotal := 0
	for i := len(commits) - 1; i >= 0; i-- {
		runningTotal += commits[i].Total
		commits[i].RunningTotal = runningTotal
	}

	err = highlightHero(commits)
	if err != nil {
		log.Fatal(err)
	}

	if chartPath != "" {
		err = chartHero(commits, chartPath)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func getTimes(commits []Commit) []time.Time {
	var times []time.Time
	for i := len(commits) - 1; i >= 0; i-- {
		times = append(times, commits[i].Date)
	}
	return times
}

func getSlocs(commits []Commit) []opts.LineData {
	items := make([]opts.LineData, 0)
	for i := len(commits) - 1; i >= 0; i-- {
		items = append(items, opts.LineData{Value: commits[i].RunningTotal, Name: commits[i].Hash})
	}
	return items
}

func chartHero(commits []Commit, fn string) error {

	// create a new bar instance
	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
	line.SetGlobalOptions(
		charts.WithDataZoomOpts(opts.DataZoom{
			Type: "inside",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Trigger:   "axis",
			TriggerOn: "mousemove|click",
			Show:      true,
			Formatter: "{b}",
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    "Less code is the best code",
			Subtitle: "https://github.com/kaihendry/lesshero",
		}),
		// label Y axis code count
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Code Count",
		}),
	)

	// draw commits
	line.SetXAxis(getTimes(commits)).AddSeries("SLOC", getSlocs(commits))

	// Where the magic happens
	f, _ := os.Create(fn)
	line.Render(f)
	return nil
}

func lessHero(path string) (commits []Commit, err error) {
	// We instantiate a new repository targeting the given path (the .git folder)
	r, err := git.PlainOpen(path)
	if err != nil {
		return
	}

	// ... retrieving the HEAD reference
	ref, err := r.Head()
	if err != nil {
		return
	}

	// ... retrieves the commit history
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return
	}

	// ... just iterates over the commits
	err = cIter.ForEach(func(c *object.Commit) error {
		// print short hash of commit
		fStats, err := c.Stats()
		total := 0
		if err != nil {
			return err
		}
		for _, fStat := range fStats {
			// log.Println(fStat.Name, fStat.Addition, fStat.Deletion)
			total += fStat.Addition - fStat.Deletion
		}
		commits = append(commits, Commit{
			Hash:   c.Hash.String()[:7],
			Author: c.Author.Name,
			Total:  total,
			Date:   c.Author.When,
		})
		return nil
	})
	return commits, err
}

func highlightHero(commits []Commit) error {
	for _, commit := range commits {
		// commitId is YYYY-MM-DD date, hash, author
		commitId := fmt.Sprintf("%s %s %s %d %d", commit.Date.Format("2006-01-02"), commit.Hash, commit.Author, commit.Total, commit.RunningTotal)
		switch {
		// color total ranges
		case commit.Total < -10:
			fmt.Println(gchalk.BrightGreen(commitId))
		case commit.Total < 0:
			fmt.Println(gchalk.Green(commitId))
		case commit.Total > 10:
			fmt.Println(gchalk.BrightRed(commitId))
		case commit.Total > 0:
			fmt.Println(gchalk.Red(commitId))
		}
	}
	return nil
}
