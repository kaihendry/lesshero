package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

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

// Open an existing repository in a specific folder.
func main() {
	// require a repoPath to a git repo using flag package
	var repoPath, jsonPath string
	flag.StringVar(&repoPath, "r", ".", "path to git repo")
	flag.StringVar(&jsonPath, "j", "output.json", "path to json output")

	flag.Parse()

	// check path exists else exit
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		log.Fatal(err)
	}

	commits, err := lessHero(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	// calculate running total, starting from the end of commits
	runningTotal := 0
	for i := len(commits) - 1; i >= 0; i-- {
		runningTotal += commits[i].Total
		commits[i].RunningTotal = runningTotal
	}

	err = highlightHero(commits)
	if err != nil {
		log.Fatal(err)
	}

	if jsonPath != "" {
		err = jsonHero(commits, jsonPath)
		if err != nil {
			log.Fatal(err)
		}
	}

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

func jsonHero(commits []Commit, fn string) error {
	// TODO: write to json
	// [
	// ["Date", "Hash", "Author", "Total", "RunningTotal"],
	// ["2020-01-01", "1234567", "John", 1, 1],
	// ]

	var data [][]string
	data = append(data, []string{"Date", "Hash", "Author", "Total", "RunningTotal"})
	for _, commit := range commits {
		data = append(data, []string{
			commit.Date.Format("2006-01-02"),
			commit.Hash,
			commit.Author,
			fmt.Sprintf("%d", commit.Total),
			fmt.Sprintf("%d", commit.RunningTotal),
		})
	}

	// create file
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	// marshal to json for https://jsfiddle.net/kaihendry/ef5n324w/

	jsonString := fmt.Sprintf("[")
	for i, e := range data {
		json, _ := json.Marshal(e)
		if i != 0 {
			jsonString += fmt.Sprintf(",")
		}
		jsonString += fmt.Sprintf("\n%s", json)
	}
	jsonString += fmt.Sprintf("\n]")

	// write jsonString to file
	_, err = f.WriteString(jsonString)

	return err
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
