package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

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
	slog.SetDefault(getLogger(os.Getenv("LOGLEVEL")))

	var chartPath string
	var startHash string
	var chartName string
	var autoOpenChart bool
	var ignoredPaths string

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		slog.Error("debug.ReadBuildInfo() failed")
		return
	}
	version := buildInfo.Main.Version
	slog.Info("lesshero", "version", version)

	flag.StringVar(&chartPath, "o", "lesshero.html", "path to html chart output")
	flag.StringVar(&startHash, "s", "", "start hash else will start from HEAD and go backwards in time")
	flag.StringVar(&chartName, "n", "", "chart title, useful when using jsonl, default is repo remote")
	flag.StringVar(&ignoredPaths, "ignore", "", "comma-separated repository-relative file paths to ignore across all history")

	flag.BoolVar(&autoOpenChart, "b", false, "auto open chart in default browser")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s [options] [git repo path]\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nhttps://github.com/kaihendry/lesshero/releases/tag/%s\n", version)
	}
	flag.Parse()
	var ignore []string
	for _, name := range strings.Split(ignoredPaths, ",") {
		if name = strings.TrimSpace(name); name != "" {
			ignore = append(ignore, name)
		}
	}

	if flag.Arg(0) != "" {
		repoPath = flag.Arg(0)
	}

	if filepath.Ext(repoPath) == ".jsonl" {
		slog.Info("reading from jsonl", "file", repoPath)
		f, err := os.Open(repoPath)
		if err != nil {
			slog.Error("opening jsonl file", "error", err)
			return
		}
		err = visualise(f, chartPath, chartName, autoOpenChart)
		if err != nil {
			slog.Error("visualising jsonl argument", "error", err)
		}
		return
	}

	slog.Info("analyzing", "repo", repoPath, "count", fmt.Sprintf("git -C %s rev-list --all --count", repoPath), "chartPath", chartPath)

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

	buf := &bytes.Buffer{}
	err = getCommits(r, startCommit, buf, ignore)
	if err != nil {
		panic(err)
	}
	err = visualise(buf, chartPath, getOrigin(r), autoOpenChart)
	if err != nil {
		slog.Error("visualising", "error", err)
	}
}

func getOrigin(r *git.Repository) (gitSrc string) {
	remotes, err := r.Remotes()
	if err != nil {
		slog.Error("getting remotes", "error", err)
		return ""
	}

	for _, remote := range remotes {
		r := remote.Config()
		if r.Name == "origin" {
			gitSrc = r.URLs[0]
		}
	}

	return gitSrc
}

func visualise(r io.Reader, chartPath string, chartName string, autoOpenChart bool) error {
	commits, err := parseLHjson(r)
	if err != nil {
		slog.Error("parsing JSON", "error", err)
		return err
	}

	for i := 0; i < len(commits); i++ {
		slog.Debug("commit", "hash", commits[i].ShortHash, "net", commits[i].Net, "date", commits[i].Date, "running total", commits[i].runningTotal)
	}

	if chartPath != "" {
		err := chartHero(commits, chartName, chartPath)
		if err != nil {
			slog.Error("creating chart", "error", err)
		}
	}
	if autoOpenChart {
		err = browser.OpenFile(chartPath)
		if err != nil {
			slog.Error("charthero open", "err", err)
			return err
		}
	}

	return nil
}

func parseLHjson(r io.Reader) ([]LHcommit, error) {
	input, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	lineCount := countLines(bytes.NewReader(input))
	commits := make([]LHcommit, 0, lineCount)
	slog.Debug("line count", "count", lineCount, "commits before appending", len(commits))

	scanner := bufio.NewScanner(bytes.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()
		commit := LHcommit{}
		err := json.Unmarshal([]byte(line), &commit)
		if err != nil {
			slog.Error("unmarshalling JSON", "line", line, "error", err)
			continue
		}
		commits = append(commits, commit)
	}

	total := 0
	for _, commit := range commits {
		slog.Debug("adding", "commit", commit.ShortHash, "net", commit.Net, "date", commit.Date)
		total += commit.Net
	}
	slog.Info("summary", "count", total, "commits", len(commits))

	slices.Reverse(commits)

	for i := 0; i < len(commits); i++ {
		slog.Debug("commit", "hash", commits[i].ShortHash, "net", commits[i].Net, "date", commits[i].Date)
		commits[i].runningTotal = commits[i].Net
		if i > 0 {
			commits[i].runningTotal += commits[i-1].runningTotal
		}
	}
	return commits, nil
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

func getCommits(r *git.Repository, commit *object.Commit, w io.Writer, ignore []string) (err error) {
	// follow the commit history via .Parent until the first commit https://github.com/go-git/go-git/issues/465#issuecomment-2121988320
	err = printJSON(commit, w, ignore)
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
		err = printJSON(commit, w, ignore)
		if err != nil {
			return err
		}
	}
	return nil
}

func printJSON(c *object.Commit, w io.Writer, ignore []string) error {
	net, err := getFstats(c, ignore)
	if err != nil {
		return fmt.Errorf("counting commit %s: %w", c.Hash, err)
	}
	lh := &LHcommit{
		ShortHash: c.Hash.String()[:7],
		Author:    c.Author.Name,
		Date:      c.Author.When,
		Email:     c.Author.Email,
		Net:       net,
	}
	return json.NewEncoder(io.MultiWriter(os.Stdout, w)).Encode(lh)
}

func getFstats(c *object.Commit, ignore []string) (total int, err error) {
	fStats, err := fileStats(c, ignore)
	if err != nil {
		return 0, err
	}
	for _, fStat := range fStats {
		total += fStat.Addition - fStat.Deletion
	}
	return total, nil
}

func fileStats(c *object.Commit, ignore []string) (object.FileStats, error) {
	if len(ignore) == 0 {
		return c.Stats()
	}
	to, err := c.Tree()
	if err != nil {
		return nil, err
	}
	var from *object.Tree
	if c.NumParents() != 0 {
		parent, err := c.Parent(0)
		if err != nil {
			return nil, err
		}
		from, err = parent.Tree()
		if err != nil {
			return nil, err
		}
	}
	// DiffTree does not detect renames: a move is a deletion plus an addition.
	// This lets each path count independently when a file enters or leaves ignore.
	changes, err := object.DiffTree(from, to)
	if err != nil {
		return nil, err
	}
	changes = slices.DeleteFunc(changes, func(change *object.Change) bool {
		return slices.Contains(ignore, change.From.Name) || slices.Contains(ignore, change.To.Name)
	})
	patch, err := changes.Patch()
	if err != nil {
		return nil, err
	}
	return patch.Stats(), nil
}
