package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestGetCommitsAndRun(t *testing.T) {
	t.Setenv("LOGLEVEL", "ERROR")
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	var head *object.Commit
	for i, content := range []string{"one\ntwo\nthree\n", "one\ntwo\n"} {
		if err := os.WriteFile(filepath.Join(dir, "code.txt"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := tree.Add("code.txt"); err != nil {
			t.Fatal(err)
		}
		hash, err := tree.Commit("change", &git.CommitOptions{Author: &object.Signature{
			Name: "Test Author", Email: "test@example.com", When: time.Unix(int64(i), 0),
		}})
		if err != nil {
			t.Fatal(err)
		}
		head, err = repo.CommitObject(hash)
		if err != nil {
			t.Fatal(err)
		}
	}
	commits, err := getCommits(head, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 || commits[0].Net != -1 || commits[1].Net != 3 || commits[0].ShortHash != head.Hash.String()[:7] {
		t.Fatalf("unexpected commits: %+v", commits)
	}
	chart := filepath.Join(t.TempDir(), "chart.html")
	var output bytes.Buffer
	if err := run([]string{"-n", "My chart", "-o", chart, dir}, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	var expected bytes.Buffer
	for _, commit := range commits {
		if err := json.NewEncoder(&expected).Encode(commit); err != nil {
			t.Fatal(err)
		}
	}
	if output.String() != expected.String() {
		t.Fatal("CLI JSONL differs from newest-first commit data")
	}
	content, err := os.ReadFile(chart)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"text":"My chart"`) {
		t.Fatal("chart title override was ignored")
	}
	if err := run([]string{"-o", "", dir}, failingWriter{}, io.Discard); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("stdout failure = %v", err)
	}
	output.Reset()
	if err := run([]string{"-s", "HEAD~1", "-o", "", dir}, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	earlier, err := parseLHjson(&output)
	if err != nil || len(earlier) != 1 || earlier[0].Net != 3 {
		t.Fatalf("start revision: commits=%+v, error=%v", earlier, err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestParseLHjson(t *testing.T) {
	author := strings.Repeat("a", 100000)
	var input bytes.Buffer
	if err := json.NewEncoder(&input).Encode(LHcommit{Author: author, Net: 3}); err != nil {
		t.Fatal(err)
	}
	commits, err := parseLHjson(&input)
	if err != nil || len(commits) != 1 || commits[0].Author != author {
		t.Fatalf("long record: count=%d, error=%v", len(commits), err)
	}
	for _, input := range []string{`not JSON`, `{"net":3`, "{\"net\":3}\ninvalid"} {
		if _, err := parseLHjson(strings.NewReader(input)); err == nil {
			t.Errorf("accepted malformed input %q", input)
		}
	}
}

func TestRunErrors(t *testing.T) {
	t.Setenv("LOGLEVEL", "ERROR")
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.jsonl")
	valid := filepath.Join(dir, "valid.jsonl")
	for path, data := range map[string]string{empty: "", valid: "{\"hash\":\"abc1234\",\"net\":3}\n"} {
		if err := os.WriteFile(path, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"empty input", []string{"-o", filepath.Join(dir, "out.html"), empty}, "no commits"},
		{"missing input", []string{filepath.Join(dir, "missing.jsonl")}, "opening JSONL"},
		{"missing repository", []string{filepath.Join(dir, "missing")}, "opening repository"},
		{"output failure", []string{"-o", dir, valid}, "creating chart"},
		{"unknown flag", []string{"-unknown"}, "flag provided but not defined"},
		{"extra arguments", []string{dir, dir}, "expected one"},
		{"browser without output", []string{"-b", "-o", "", valid}, "requires a chart"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.args, io.Discard, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
	var help bytes.Buffer
	if err := run([]string{"-h"}, io.Discard, &help); err != nil || !strings.Contains(help.String(), "Usage:") {
		t.Fatalf("help failed: %v", err)
	}
	t.Setenv("LOGLEVEL", "invalid")
	if err := run(nil, io.Discard, io.Discard); err == nil || !strings.Contains(err.Error(), "LOGLEVEL") {
		t.Fatalf("invalid log level: %v", err)
	}
}
