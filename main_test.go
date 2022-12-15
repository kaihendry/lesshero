// bench mark lessHero
package main

import (
	"testing"
)

func BenchmarkHero(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lessHero(".")
	}
}

func Test_lessHeroOrder(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
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
			// check wantCommits are in order
			for i := 0; i < len(gotCommits)-1; i++ {
				if gotCommits[i].date.After(gotCommits[i+1].date) {
					t.Errorf("lessHero() commits not in order")
				}
			}
		})
	}
}
