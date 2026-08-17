package redirect

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/Debian/debiman/internal/proto"
	"github.com/golang/protobuf/proto"
)

var (
	benchSuites   = []string{"jessie", "wheezy", "testing", "unstable", "experimental"}
	benchLangs    = []string{"en", "fr", "es"}
	benchSections = []string{"1", "3", "3edit", "5", "8"}
)

// benchIndex builds an index which resembles the real one served by
// debiman-auxserver: one entry per (name, suite, section, language)
// combination.
func benchIndex(names int) Index {
	idx := Index{
		Entries:  make(map[string][]IndexEntry, names),
		Suites:   make(map[string]string, len(benchSuites)),
		Langs:    make(map[string]bool, len(benchLangs)),
		Sections: make(map[string]bool, len(benchSections)),
	}
	for _, suite := range benchSuites {
		idx.Suites[suite] = suite
	}
	idx.Suites["stable"] = "jessie"
	idx.Suites["oldstable"] = "wheezy"
	for _, lang := range benchLangs {
		idx.Langs[lang] = true
	}
	for _, section := range benchSections {
		idx.Sections[section] = true
	}
	idx.Sections["0"] = true

	for i := 0; i < names; i++ {
		name := fmt.Sprintf("manpage-%d", i)
		entries := make([]IndexEntry, 0, len(benchSuites)*len(benchSections)*len(benchLangs))
		for _, suite := range benchSuites {
			for _, section := range benchSections {
				for _, lang := range benchLangs {
					entries = append(entries, IndexEntry{
						Name:      name,
						Suite:     suite,
						Binarypkg: fmt.Sprintf("binarypkg-%d", i%128),
						Section:   section,
						Language:  lang,
					})
				}
			}
		}
		idx.Entries[name] = entries
	}
	return idx
}

// benchPaths are the shapes of URLs which manpages.debian.org sees:
// underspecified, fully qualified, legacy manpages.debian.org paths and
// raw manpage requests.
var benchPaths = []string{
	"/i3",
	"/i3.5",
	"/i3.5.fr",
	"/jessie/i3-wm/i3.1.en",
	"/jessie/i3-wm/i3.1.en.gz",
	"/testing/i3",
	"/man1/i3",
	"/fr/man5/i3",
	"/i3(1)",
}

// Redirect is the hot path of debiman-auxserver: it runs once per HTTP
// request.
func BenchmarkRedirect(b *testing.B) {
	// Redirect() logs the parsed request, which would otherwise dominate
	// the measurement (and spam the benchmark output).
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	requests := make([]*http.Request, len(benchPaths))
	for idx, path := range benchPaths {
		u, err := url.Parse("http://manpages.debian.org" + path)
		if err != nil {
			b.Fatal(err)
		}
		requests[idx] = &http.Request{URL: u, Header: http.Header{}}
	}

	for idx, path := range benchPaths {
		r := requests[idx]
		b.Run(path, func(b *testing.B) {
			for b.Loop() {
				if _, err := testIdx.Redirect(r); err != nil {
					if _, ok := err.(*NotFoundError); !ok {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

// BenchmarkRedirectAcceptLanguage additionally exercises the
// language.Matcher based negotiation.
func BenchmarkRedirectAcceptLanguage(b *testing.B) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	u, err := url.Parse("http://manpages.debian.org/i3")
	if err != nil {
		b.Fatal(err)
	}
	header := http.Header{}
	header.Set("Accept-Language", "fr-CH, fr;q=0.9, en;q=0.8, de;q=0.7, *;q=0.5")
	r := &http.Request{URL: u, Header: header}

	for b.Loop() {
		if _, err := testIdx.Redirect(r); err != nil {
			b.Fatal(err)
		}
	}
}

// Narrow does the actual work of picking the best manpage out of all
// candidates.
func BenchmarkNarrow(b *testing.B) {
	idx := benchIndex(1)
	entries := idx.Entries["manpage-0"]
	var ref IndexEntry

	table := []struct {
		name     string
		template IndexEntry
		lang     string
	}{
		{name: "empty", template: IndexEntry{}},
		{name: "suite-only", template: IndexEntry{Suite: "testing"}},
		{name: "subsection", template: IndexEntry{Section: "3edit"}},
		{name: "accept-language", template: IndexEntry{}, lang: "fr-CH, fr;q=0.9, en;q=0.8"},
		{name: "fully-qualified", template: IndexEntry{
			Suite:     "testing",
			Binarypkg: "binarypkg-0",
			Section:   "1",
			Language:  "fr",
		}},
	}

	for _, entry := range table {
		b.Run(entry.name, func(b *testing.B) {
			for b.Loop() {
				if got := idx.Narrow(entry.lang, entry.template, ref, entries); len(got) == 0 {
					b.Fatalf("Narrow(%v) unexpectedly returned no entries", entry.template)
				}
			}
		})
	}
}

// split parses the request path into its components.
func BenchmarkSplit(b *testing.B) {
	idx := benchIndex(1)
	for b.Loop() {
		for _, path := range benchPaths {
			idx.split(path)
		}
	}
}

// IndexFromProto is called on startup (and on every index reload) of
// debiman-auxserver, with an index containing millions of entries.
func BenchmarkIndexFromProto(b *testing.B) {
	idx := benchIndex(500)

	pbIdx := &pb.Index{Suite: idx.Suites}
	for _, entries := range idx.Entries {
		for _, e := range entries {
			pbIdx.Entry = append(pbIdx.Entry, &pb.IndexEntry{
				Name:      e.Name,
				Suite:     e.Suite,
				Binarypkg: e.Binarypkg,
				Section:   e.Section,
				Language:  e.Language,
			})
		}
	}
	for lang := range idx.Langs {
		pbIdx.Language = append(pbIdx.Language, lang)
	}
	for section := range idx.Sections {
		pbIdx.Section = append(pbIdx.Section, section)
	}

	idxb, err := proto.Marshal(pbIdx)
	if err != nil {
		b.Fatal(err)
	}
	path := filepath.Join(b.TempDir(), "index.pb")
	if err := os.WriteFile(path, idxb, 0644); err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(idxb)))
	for b.Loop() {
		got, err := IndexFromProto(path)
		if err != nil {
			b.Fatal(err)
		}
		if len(got.Entries) == 0 {
			b.Fatal("IndexFromProto unexpectedly returned an empty index")
		}
	}
}
