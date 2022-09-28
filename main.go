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
	"github.com/schollz/progressbar/v3"
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
	repoPath := "."
	flag.StringVar(&chartPath, "c", "", "path to html chart output")
	highlight := flag.Bool("hl", false, "highlight")
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
		runningTotal += commits[i].total
		commits[i].runningTotal = runningTotal
	}

	if *highlight {
		highlightHero(commits)
	}

	if chartPath != "" {
		err = chartHero(commits, chartPath)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func getTimes(commits []Commit) (times []string) {
	for i := len(commits) - 1; i >= 0; i-- {
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

func chartHero(commits []Commit, fn string) error {

	line := charts.NewLine()
	// set some global options like Title/Legend/ToolTip or anything else
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
		charts.WithTitleOpts(opts.Title{Title: "https://github.com/kaihendry/lesshero"}),
		// label Y axis code count
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Code Count",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Rotate: 20,
			},
		}),
	)

	// draw commits
	line.SetXAxis(getTimes(commits)).AddSeries("SLOC", getSlocs(commits))

	dynamicFn := fmt.Sprintf(`goecharts_%s.on('click', function (params) {   navigator.clipboard.writeText(params.name); console.log(params.name, "copied to clipboard"); });`, line.ChartID)
	line.AddJSFuncs(dynamicFn)

	f, err := os.Create(fn)
	if err != nil {
		return err
	}
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

	// git rev-list HEAD --count
	count := 0
	err = cIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	if err != nil {
		return
	}

	// log.Printf("Total commits: %d", count)

	cIter, err = r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return
	}

	bar := progressbar.Default(int64(count))
	commits = make([]Commit, count)
	countIndex := 0

	// print length of commits
	// log.Printf("Commits: %d", len(commits))

	err = cIter.ForEach(func(c *object.Commit) error {
		bar.Add(1)
		fStats, err := c.Stats()
		total := 0
		if err != nil {
			return err
		}
		for _, fStat := range fStats {
			// log.Println(fStat.Name, fStat.Addition, fStat.Deletion)
			total += fStat.Addition - fStat.Deletion
		}
		commits[countIndex] = Commit{
			hash:   c.Hash.String()[:7],
			author: c.Author.Name,
			total:  total,
			date:   c.Author.When,
		}
		countIndex++
		return nil
	})
	return commits, err
}

func highlightHero(commits []Commit) {
	for _, commit := range commits {
		// commitId is YYYY-MM-DD date, hash, author
		commitId := fmt.Sprintf("%s %s %s %d %d", commit.date.Format("2006-01-02"), commit.hash, commit.author, commit.total, commit.runningTotal)
		switch {
		// color total ranges
		case commit.total < -10:
			fmt.Println(gchalk.BrightGreen(commitId))
		case commit.total < 0:
			fmt.Println(gchalk.Green(commitId))
		case commit.total > 10:
			fmt.Println(gchalk.BrightRed(commitId))
		case commit.total > 0:
			fmt.Println(gchalk.Red(commitId))
		}
	}
}
