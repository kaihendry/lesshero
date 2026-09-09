package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestIgnoreHistory(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	// nil contents delete a path. Deleting and adding in one commit models a rename.
	steps := []struct {
		name  string
		files map[string]*string
		all   int
		both  int
		new   int
	}{
		{"initial", map[string]*string{
			"main.go":        textContent("package main\n// code\n"),
			"all.json":       textContent("one\ntwo\nthree\n"),
			"other/all.json": textContent("still counted\n"),
			"binary.dat":     textContent("\x00one\ntwo\n"),
		}, 6, 3, 6},
		{"update data", map[string]*string{
			"all.json": textContent("one\ntwo\nthree\nfour\n"),
		}, 7, 3, 7},
		{"move and edit data", map[string]*string{
			"all.json":        nil,
			"static/all.json": textContent("one\ntwo\nthree\nfour\nfive\n"),
		}, 8, 3, 3},
		{"update code", map[string]*string{
			"main.go": textContent("package main\n// code\n// more code\n"),
		}, 9, 4, 4},
		{"move out of ignore", map[string]*string{
			"static/all.json": nil,
			"visible.json":    textContent("one\ntwo\nthree\nfour\nfive\n"),
		}, 9, 9, 9},
		{"move into ignore", map[string]*string{
			"visible.json":    nil,
			"static/all.json": textContent("one\ntwo\nthree\nfour\nfive\nsix\n"),
		}, 10, 4, 4},
		{"delete data", map[string]*string{"static/all.json": nil}, 4, 4, 4},
		{"delete code", map[string]*string{"main.go": nil, "other/all.json": nil}, 0, 0, 0},
		{"restore old path", map[string]*string{"all.json": textContent("one\ntwo\n")}, 2, 0, 2},
		{"delete old path", map[string]*string{"all.json": nil}, 0, 0, 0},
	}
	var head *object.Commit
	for i, step := range steps {
		for name, content := range step.files {
			if content == nil {
				if _, err := worktree.Remove(name); err != nil {
					t.Fatal(err)
				}
				continue
			}
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(*content), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := worktree.Add(name); err != nil {
				t.Fatal(err)
			}
		}
		hash, err := worktree.Commit(step.name, &git.CommitOptions{Author: &object.Signature{
			Name: "Test", Email: "test@example.com", When: time.Unix(int64(i), 0),
		}})
		if err != nil {
			t.Fatal(err)
		}
		head, err = repo.CommitObject(hash)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		ignore []string
		want   func(int) int
	}{
		{"default", nil, func(i int) int { return steps[i].all }},
		{"unmatched", []string{"missing.json"}, func(i int) int { return steps[i].all }},
		{"both names", []string{"all.json", "static/all.json"}, func(i int) int { return steps[i].both }},
		{"new name only", []string{"static/all.json"}, func(i int) int { return steps[i].new }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := getCommits(repo, head, &buf, tc.ignore); err != nil {
				t.Fatal(err)
			}
			commits, err := parseLHjson(&buf)
			if err != nil {
				t.Fatal(err)
			}
			if len(commits) != len(steps) {
				t.Fatalf("got %d commits, want %d", len(commits), len(steps))
			}
			for i, commit := range commits {
				if got, want := commit.runningTotal, tc.want(i); got != want {
					t.Errorf("%s: total = %d, want %d", steps[i].name, got, want)
				}
			}
		})
	}
}

func textContent(s string) *string { return &s }
