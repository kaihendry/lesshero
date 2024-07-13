package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"runtime/debug"
	"slices"

	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/browser"
)

type LHcommit struct {
	ShortHash    string    `json:"hash"`
	Author       string    `json:"author"`
	Date         time.Time `json:"date"`
	Email        string    `json:"email"`
	Net          int       `json:"net"`
	runningTotal int
}

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

func main() {
	// cpuFile, err := os.Create("cpu.prof")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer cpuFile.Close()
	// if err := pprof.StartCPUProfile(cpuFile); err != nil {
	// 	log.Fatal("could not start CPU profile: ", err)
	// }
	// defer pprof.StopCPUProfile()
	slog.SetDefault(getLogger(os.Getenv("LOGLEVEL")))

	var chartPath string
	var startHash string
	var chartName string
	var autoOpenChart bool

	version, dirty := GitCommit()
	slog.Info("lh version", version, dirty)

	flag.StringVar(&chartPath, "o", "lesshero.html", "path to html chart output")
	flag.StringVar(&startHash, "s", "", "start hash else will start from HEAD and go backwards in time")
	flag.StringVar(&chartName, "n", "lesshero", "chartName")

	flag.BoolVar(&autoOpenChart, "b", false, "auto open chart in default browser")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s [options] [git repo path]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nhttps://github.com/kaihendry/lesshero/commit/%s\n", version)
	}
	flag.Parse()

	if isInputFromStdin() {
		slog.Info("Input from stdin")
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			return
		}

		lineCount := countLines(bytes.NewReader(input))
		commits := make([]LHcommit, 0, lineCount)
		slog.Info("line count", "count", lineCount, "commits before appending", len(commits))

		scanner := bufio.NewScanner(bytes.NewReader(input))

		for scanner.Scan() {
			line := scanner.Text()
			commit := LHcommit{}
			err := json.Unmarshal([]byte(line), &commit)
			if err != nil {
				slog.Error("Error unmarshalling JSON", "line", line, "error", err)
				continue
			}
			commits = append(commits, commit)
		}

		// calculate total
		total := 0
		for _, commit := range commits {
			slog.Debug("adding", "commit", commit.ShortHash, "net", commit.Net, "date", commit.Date)
			total += commit.Net
		}
		slog.Info("total", "count", total, "commit count", len(commits))

		// reverse slice
		slices.Reverse(commits)

		// calculate running total
		for i := 0; i < len(commits); i++ {
			slog.Debug("commit", "hash", commits[i].ShortHash, "net", commits[i].Net, "date", commits[i].Date)
			commits[i].runningTotal = commits[i].Net
			if i > 0 {
				commits[i].runningTotal += commits[i-1].runningTotal
			}
		}

		for i := 0; i < len(commits); i++ {
			slog.Debug("commit", "hash", commits[i].ShortHash, "net", commits[i].Net, "date", commits[i].Date, "running total", commits[i].runningTotal)
		}

		// write chart if requested
		if chartPath != "" {
			err := chartHero(commits, chartName, chartPath, total)
			if err != nil {
				slog.Error("Error creating chart", "error", err)
			}
		}
		if autoOpenChart {
			err = browser.OpenFile(chartPath)
			if err != nil {
				slog.Error("charthero open", "err", err)
				return
			}
		}

		return
	}

	if flag.Arg(0) != "" {
		repoPath = flag.Arg(0)
	}

	// read the repository from os.Args
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		panic(err)
	}
	// show the current branch
	ref, err := r.Head()
	if err != nil {
		panic(err)
	}
	slog.Info("default branch", "branch", ref.Name().String())
	startCommit, err := r.CommitObject(ref.Hash())
	if err != nil {
		panic(err)
	}
	if startHash != "" {
		rev := plumbing.Revision(startHash)
		hash, err := r.ResolveRevision(rev)
		if err != nil {
			panic(err)
		}
		startCommit, err = r.CommitObject(*hash)
		if err != nil {
			panic(err)
		}
	}
	slog.Info("start commit", "hash", startCommit.Hash.String(), "startHash", startHash)
	err = getCommits(r, startCommit)
	if err != nil {
		panic(err)
	}
}

func countLines(r io.Reader) int {
	scanner := bufio.NewScanner(r)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error counting lines: %v\n", err)
	}
	return lineCount
}

func isInputFromStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		return false
	} else {
		return true
	}
}

func GitCommit() (commit string, dirty bool) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.modified":
			dirty = setting.Value == "true"
		case "vcs.revision":
			commit = setting.Value
		}
	}
	return
}

func getCommits(r *git.Repository, commit *object.Commit) (err error) {
	// follow the commit history via .Parent until the first commit https://github.com/go-git/go-git/issues/465#issuecomment-2121988320
	err = printJSON(commit)
	if err != nil {
		return err
	}

	for {
		if commit.NumParents() == 0 {
			slog.Warn("no more parents", "hash", commit.Hash.String())
			break
		}
		commit, err = commit.Parents().Next()
		if err != nil {
			return err
		}
		err = printJSON(commit)
		if err != nil {
			return err
		}
	}
	return nil
}

func printJSON(c *object.Commit) error {
	lh := &LHcommit{
		ShortHash: c.Hash.String()[:7],
		Author:    c.Author.Name,
		Date:      c.Author.When,
		Email:     c.Author.Email,
		Net:       getFstats(c),
	}
	// json encoder
	jsonData, err := json.Marshal(lh)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func getFstats(c *object.Commit) (total int) {
	fStats, err := c.Stats()
	if err != nil {
		return 0
	}
	for _, fStat := range fStats {
		total += fStat.Addition - fStat.Deletion
	}
	return total
}
