package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jwalton/gchalk"
)

// Open an existing repository in a specific folder.
func main() {
	// require a path to a git repo using flag package
	var path string
	flag.StringVar(&path, "r", ".", "path to git repo")
	flag.Parse()

	// check path exists else exit
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal(err)
	}

	// We instantiate a new repository targeting the given path (the .git folder)
	r, err := git.PlainOpen(path)
	if err != nil {
		log.Fatal(err)
	}

	// ... retrieving the HEAD reference
	ref, err := r.Head()
	if err != nil {
		log.Fatal(err)
	}

	// ... retrieves the commit history
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		log.Fatal(err)
	}

	// ... just iterates over the commits
	err = cIter.ForEach(func(c *object.Commit) error {
		// print short hash of commit
		fStats, err := c.Stats()
		total := 0
		if err != nil {
			log.Fatal(err)
		}
		for _, fStat := range fStats {
			// fmt.Println(fStat.Name, fStat.Addition, fStat.Deletion)
			total += fStat.Addition - fStat.Deletion
		}
		// commitId is YYYY-MM-DD date, hash, author
		commitId := fmt.Sprintf("%s %s %s %d", c.Author.When.Format("2006-01-02"), c.Hash.String()[:7], c.Author.Email, total)
		switch {
		// color total ranges
		case total < -10:
			fmt.Println(gchalk.BrightGreen(commitId))
		case total < 0:
			fmt.Println(gchalk.BrightYellow(commitId))
		case total > 10:
			fmt.Println(gchalk.BrightRed(commitId))
		case total > 0:
			fmt.Println(gchalk.Red(commitId))
		}

		return nil
	})
}
