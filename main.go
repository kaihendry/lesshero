package main

import (
	"encoding/json"
	"errors"
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
	ShortHash string    `json:"hash"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
	Email     string    `json:"email"`
	Net       int       `json:"net"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		slog.Error("lesshero", "error", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	level := slog.LevelInfo
	if value := os.Getenv("LOGLEVEL"); value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			return fmt.Errorf("invalid LOGLEVEL: %w", err)
		}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: level})))

	version := "(devel)"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	flags := flag.NewFlagSet("lesshero", flag.ContinueOnError)
	// main reports errors once; print usage only when help is requested.
	flags.SetOutput(io.Discard)
	chartPath := flags.String("o", "lesshero.html", "path to html chart output")
	startHash := flags.String("s", "HEAD", "start hash else will start from HEAD and go backwards in time")
	chartName := flags.String("n", "", "chart title, useful when using jsonl, default is repo remote")
	ignoredPaths := flags.String("ignore", "", "comma-separated repository-relative file paths to ignore across all history")
	autoOpenChart := flags.Bool("b", false, "auto open chart in default browser")
	flags.Usage = func() {
		_, _ = fmt.Fprintln(flags.Output(), "Usage: lesshero [options] [git repo path or JSONL file]")
		flags.PrintDefaults()
		_, _ = fmt.Fprintf(flags.Output(), "\nhttps://github.com/kaihendry/lesshero/releases/tag/%s\n", version)
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(stderr)
			flags.Usage()
			return nil
		}
		return err
	}
	if flags.NArg() > 1 {
		return errors.New("expected one repository path or JSONL file")
	}
	if *autoOpenChart && *chartPath == "" {
		return errors.New("-b requires a chart output path")
	}
	repoPath := "."
	if flags.NArg() == 1 {
		repoPath = flags.Arg(0)
	}
	slog.Info("lesshero", "version", version)

	var commits []LHcommit
	if filepath.Ext(repoPath) == ".jsonl" {
		f, err := os.Open(repoPath)
		if err != nil {
			return fmt.Errorf("opening JSONL: %w", err)
		}
		commits, err = parseLHjson(f)
		err = errors.Join(err, f.Close())
		if err != nil {
			return fmt.Errorf("reading JSONL: %w", err)
		}
	} else {
		r, err := git.PlainOpen(repoPath)
		if err != nil {
			return fmt.Errorf("opening repository: %w", err)
		}
		if *startHash == "" {
			*startHash = "HEAD"
		}
		hash, err := r.ResolveRevision(plumbing.Revision(*startHash))
		if err != nil {
			return fmt.Errorf("resolving %s: %w", *startHash, err)
		}
		head, err := r.CommitObject(*hash)
		if err != nil {
			return fmt.Errorf("reading start commit: %w", err)
		}
		var ignore []string
		for _, name := range strings.Split(*ignoredPaths, ",") {
			if name = strings.TrimSpace(name); name != "" {
				ignore = append(ignore, name)
			}
		}
		commits, err = getCommits(head, ignore)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(stdout)
		for _, commit := range commits {
			if err := encoder.Encode(commit); err != nil {
				return fmt.Errorf("writing JSONL: %w", err)
			}
		}
		if *chartName == "" {
			*chartName = getOrigin(r)
		}
	}

	slices.Reverse(commits)
	if *chartPath != "" {
		if err := chartHero(commits, *chartName, *chartPath); err != nil {
			return fmt.Errorf("creating chart: %w", err)
		}
	}
	if *autoOpenChart {
		if err := browser.OpenFile(*chartPath); err != nil {
			return fmt.Errorf("opening chart: %w", err)
		}
	}
	return nil
}

func getOrigin(r *git.Repository) string {
	remote, err := r.Remote("origin")
	if err != nil || len(remote.Config().URLs) == 0 {
		return ""
	}
	return remote.Config().URLs[0]
}

// parseLHjson reads commits in their original, newest-first order.
func parseLHjson(r io.Reader) ([]LHcommit, error) {
	var commits []LHcommit
	decoder := json.NewDecoder(r)
	for {
		var commit LHcommit
		err := decoder.Decode(&commit)
		if errors.Is(err, io.EOF) {
			return commits, nil
		}
		if err != nil {
			return nil, fmt.Errorf("commit %d: %w", len(commits)+1, err)
		}
		commits = append(commits, commit)
	}
}

// getCommits follows first parents, returning commits newest first.
func getCommits(commit *object.Commit, ignore []string) ([]LHcommit, error) {
	var commits []LHcommit
	for {
		net, err := getFstats(commit, ignore)
		if err != nil {
			return nil, fmt.Errorf("counting commit %s: %w", commit.Hash, err)
		}
		commits = append(commits, LHcommit{
			ShortHash: commit.Hash.String()[:7],
			Author:    commit.Author.Name,
			Date:      commit.Author.When,
			Email:     commit.Author.Email,
			Net:       net,
		})
		if commit.NumParents() == 0 {
			return commits, nil
		}
		commit, err = commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("reading parent: %w", err)
		}
	}
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
