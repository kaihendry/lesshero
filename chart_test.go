package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestChartAttributionAndAuthors(t *testing.T) {
	// Commits on the same date must still show their own authors and counts.
	date := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	commits := []LHcommit{
		{ShortHash: "abc1234", Author: "Alice", Date: date, Net: 10},
		{ShortHash: "def5678", Author: `Bob "Hero" <script>alert('x')</script> & Co`, Date: date, Net: -3},
	}
	for _, title := range []string{"example/repo", ""} {
		t.Run(title, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "chart.html")
			if err := chartHero(commits, title, path); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{`"subtext":"Made with Less Hero"`, `"sublink":"https://github.com/kaihendry/lesshero"`} {
				if !strings.Contains(string(content), want) {
					t.Errorf("missing attribution: %s", want)
				}
			}
			match := regexp.MustCompile(`const tooltips_\w+ = (\[.*\]);`).FindSubmatch(content)
			if len(match) != 2 {
				t.Fatal("missing tooltip data")
			}
			var tooltips []string
			if err := json.Unmarshal(match[1], &tooltips); err != nil {
				t.Fatal(err)
			}
			want := []string{
				"2026-09-09 · abc1234<br/>Alice<br/>10 SLOC (+10)",
				"2026-09-09 · def5678<br/>Bob &#34;Hero&#34; &lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt; &amp; Co<br/>7 SLOC (-3)",
			}
			if len(tooltips) != len(want) {
				t.Fatalf("got %d tooltips, want %d", len(tooltips), len(want))
			}
			for i := range want {
				if tooltips[i] != want[i] {
					t.Errorf("tooltip %d = %q, want %q", i, tooltips[i], want[i])
				}
			}
		})
	}
}
