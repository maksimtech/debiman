package sitemap

import (
	"fmt"
	"io"
	"testing"
	"time"
)

func benchContents(n int) map[string]time.Time {
	contents := make(map[string]time.Time, n)
	modTime := time.Date(2024, 5, 17, 13, 37, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		contents[fmt.Sprintf("binarypkg-%d", i)] = modTime.Add(time.Duration(i) * time.Minute)
	}
	return contents
}

// One sitemap is written per suite, listing every binary package which
// ships manpages (tens of thousands for Debian unstable).
func BenchmarkWriteTo(b *testing.B) {
	for _, n := range []int{100, 10000} {
		contents := benchContents(n)
		b.Run(fmt.Sprintf("%d-packages", n), func(b *testing.B) {
			for b.Loop() {
				if err := WriteTo(io.Discard, "https://manpages.debian.org", contents); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkWriteIndexTo(b *testing.B) {
	contents := benchContents(20)
	for b.Loop() {
		if err := WriteIndexTo(io.Discard, "https://manpages.debian.org", contents); err != nil {
			b.Fatal(err)
		}
	}
}
