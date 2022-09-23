// bench mark lessHero
package main

import "testing"

func BenchmarkHero(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lessHero(".")
	}
}
