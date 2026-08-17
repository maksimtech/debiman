package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Debian/debiman/internal/manpage"
)

func mustParseFromServingPathBench(b *testing.B, path string) *manpage.Meta {
	m, err := manpage.FromServingPath("/srv/man", path)
	if err != nil {
		b.Fatal(err)
	}
	return m
}

// benchVersions is representative of the “versions” slice which debiman
// passes to the renderer: the same manpage in multiple suites, sections,
// languages and binary packages.
func benchVersions(b *testing.B) (*manpage.Meta, []*manpage.Meta) {
	paths := []string{
		"jessie/cron/crontab.1.en",
		"jessie/cron/crontab.5.en",
		"jessie/manpages-fr-extra/crontab.1.fr",
		"jessie/manpages-fr-extra/crontab.5.fr",
		"jessie/manpages-ja/crontab.5.ja",
		"jessie/systemd-cron/crontab.5.en",
		"jessie/bcron-run/crontab.5.en",
		"testing/cron/crontab.1.en",
		"testing/cron/crontab.5.en",
		"testing/manpages-fr-extra/crontab.5.fr",
		"testing/manpages-de/crontab.5.de",
		"testing/systemd-cron/crontab.5.en",
		"unstable/cron/crontab.1.en",
		"unstable/cron/crontab.5.en",
		"unstable/manpages-fr-extra/crontab.5.fr",
		"unstable/manpages-pt-br/crontab.5.pt_BR",
	}
	versions := make([]*manpage.Meta, len(paths))
	for idx, path := range paths {
		versions[idx] = mustParseFromServingPathBench(b, path)
	}
	return versions[0], versions
}

// bestLanguageMatch is called for every cross-reference of every rendered
// manpage, which makes it one of the hottest functions of the rendering
// stage.
func BenchmarkBestLanguageMatch(b *testing.B) {
	current, versions := benchVersions(b)
	options := make([]*manpage.Meta, len(versions))

	for b.Loop() {
		copy(options, versions)
		if best := bestLanguageMatch(current, options); best == nil {
			b.Fatal("bestLanguageMatch unexpectedly returned nil")
		}
	}
}

// benchRenderedPage builds a gzipped, already-rendered manpage as it is
// found in the serving directory of a previous debiman run.
func benchRenderedPage(b *testing.B) string {
	fragment, err := os.ReadFile("../../testdata/i3lock.html")
	if err != nil {
		b.Fatal(err)
	}

	var page bytes.Buffer
	page.WriteString("<html><body>\n<div id=\"header\">\n")
	for _, section := range []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "SEE ALSO"} {
		fmt.Fprintf(&page, "  <a class=\"toclink\" href=\"#%s\">%s</a>\n", section, section)
	}
	page.WriteString("</div>\n")
	page.Write(fragment)
	page.WriteString("</div>\n</div>\n<div id=\"footer\">\nfooter\n</div>\n</body></html>\n")

	path := filepath.Join(b.TempDir(), "crontab.1.en.html.gz")
	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	gzipw, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := gzipw.Write(page.Bytes()); err != nil {
		b.Fatal(err)
	}
	if err := gzipw.Close(); err != nil {
		b.Fatal(err)
	}
	if err := f.Close(); err != nil {
		b.Fatal(err)
	}
	return path
}

// reuse extracts the mandoc output of an already-rendered manpage, which
// is what makes incremental debiman runs fast.
func BenchmarkReuse(b *testing.B) {
	path := benchRenderedPage(b)

	for b.Loop() {
		doc, toc, err := reuse(path)
		if err != nil {
			b.Fatal(err)
		}
		if doc == "" || len(toc) == 0 {
			b.Fatalf("reuse(%q) unexpectedly returned doc=%d bytes, toc=%d entries", path, len(doc), len(toc))
		}
	}
}

func benchRenderJob(b *testing.B) renderJob {
	meta, versions := benchVersions(b)
	page := benchRenderedPage(b)
	xref := make(map[string][]*manpage.Meta)
	for _, v := range versions {
		xref[v.Name] = append(xref[v.Name], v)
	}
	return renderJob{
		dest:     filepath.Join(b.TempDir(), "crontab.1.en.html.gz"),
		src:      page,
		meta:     meta,
		versions: versions,
		xref:     xref,
		modTime:  time.Date(2024, 5, 17, 13, 37, 0, 0, time.UTC),
		reuse:    page,
	}
}

// rendermanpageprep gathers all the metadata (suites, sections, binary
// packages, languages) which the manpage template needs.
func BenchmarkRendermanpagePrep(b *testing.B) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	job := benchRenderJob(b)

	for b.Loop() {
		if _, _, err := rendermanpageprep(nil, job); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRendermanpageExecute measures only the html/template execution,
// which runs once per manpage (~200k times for all of Debian).
func BenchmarkRendermanpageExecute(b *testing.B) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	job := benchRenderJob(b)
	t, data, err := rendermanpageprep(nil, job)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		if err := t.Execute(io.Discard, data); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRendermanpage covers the whole per-manpage pipeline: reusing the
// previous output, preparing the data, executing the template and writing
// the gzip-compressed result to disk.
func BenchmarkRendermanpage(b *testing.B) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	job := benchRenderJob(b)
	gzipw, err := gzip.NewWriterLevel(nil, gzip.BestCompression)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		if _, err := rendermanpage(gzipw, nil, job); err != nil {
			b.Fatal(err)
		}
	}
}

// renderPkgindex writes one index page per binary package.
func BenchmarkRenderPkgindex(b *testing.B) {
	_, versions := benchVersions(b)
	manpageByName := make(map[string]*manpage.Meta, len(versions))
	for _, v := range versions {
		manpageByName[v.Name+"."+v.Section+"."+v.Language] = v
	}
	dest := filepath.Join(b.TempDir(), "index.html.gz")

	for b.Loop() {
		if err := renderPkgindex(dest, manpageByName); err != nil {
			b.Fatal(err)
		}
	}
}
