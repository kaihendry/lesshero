package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestGetCommits(t *testing.T) {
	// use current git repository for testing
	repo, err := git.PlainOpen(".")
	if err != nil {
		t.Fatal(err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatal(err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer

	err = getCommits(repo, commit, &buf)
	if err != nil {
		t.Fatal(err)
	}

	// parse the output with parseLHjson
	commits, err := parseLHjson(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	// check commits first commit contains the initial commit of bbed8b3 with net 167
	if commits[0].ShortHash != "bbed8b3" {
		t.Errorf("Expected bbed8b3, got %s", commits[0].ShortHash)
	}
	if commits[0].Net != 167 {
		t.Errorf("Expected 167, got %d", commits[0].Net)
	}
}
