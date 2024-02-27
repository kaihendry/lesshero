package main

import (
	"testing"
)

func Test_lessHeroOrder(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCommits, _, err := lessHero(".")
			if (err != nil) != tt.wantErr {
				t.Errorf("lessHero() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, c := range gotCommits {
				t.Log(c.hash)
			}

			if gotCommits[0].hash != "bbed8b3" {
				t.Errorf("lessHero() = %v, want %v", gotCommits[0].hash, "bbed8b3")
			}

			for i := 0; i < len(gotCommits)-1; i++ {
				if gotCommits[i].date.After(gotCommits[i+1].date) {
					t.Errorf("lessHero() = %v %v, is not after %v %v", gotCommits[i].date, gotCommits[i].hash, gotCommits[i+1].date, gotCommits[i+1].hash)
				}
			}
		})
	}
}
