package manpage

import "testing"

// benchPkg is representative of the metadata debiman carries around
// while walking the file list of a binary package.
var benchPkg = PkgMeta{
	Component: "main",
	Filename:  "pool/main/i/i3-wm/i3-wm_4.23-1_amd64.deb",
	Sourcepkg: "i3-wm",
	Binarypkg: "i3-wm",
	Suite:     "testing",
}

// benchManPaths covers the interesting shapes of manpage paths: plain
// English pages, translated pages, pages with a codeset or a modifier in
// the language directory, pages without the .gz suffix and subsections.
var benchManPaths = []string{
	"man1/i3.1.gz",
	"man3/el_init.3",
	"man3/editline.3edit",
	"man5/i3.5.gz",
	"fr/man1/i3.1.gz",
	"pt_BR/man1/deja-dup.1.gz",
	"bg.UTF-8/man6/hex-a-hop.6.gz",
	"ca@valencia/man1/deja-dup.1.gz",
}

// FromManPath is called once per file of every binary package which ships
// manpages, i.e. hundreds of thousands of times for a full run.
func BenchmarkFromManPath(b *testing.B) {
	for _, path := range benchManPaths {
		b.Run(path, func(b *testing.B) {
			for b.Loop() {
				if _, err := FromManPath(path, &benchPkg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFromManPathMixed(b *testing.B) {
	for b.Loop() {
		for _, path := range benchManPaths {
			if _, err := FromManPath(path, &benchPkg); err != nil {
				b.Fatal(err)
			}
		}
	}
}

var benchServingPaths = []string{
	"/srv/man/testing/i3-wm/i3.1.en",
	"/srv/man/testing/i3-wm/i3.1.fr.gz",
	"/srv/man/jessie/manpages-fr-extra/crontab.5.fr",
	"/srv/man/testing/kde-l10n-sr/ark.1.sr@latin",
}

// FromServingPath is called for every already-rendered manpage when
// debiman inspects the serving directory of an incremental run.
func BenchmarkFromServingPath(b *testing.B) {
	for b.Loop() {
		for _, path := range benchServingPaths {
			if _, err := FromServingPath("/srv/man", path); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkServingPath(b *testing.B) {
	m, err := FromManPath("fr/man1/i3.1.gz", &benchPkg)
	if err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		sink = m.ServingPath()
	}
}

// SameBinary is called in the inner loop of the rendering stage, once per
// candidate version of a manpage.
func BenchmarkSameBinary(b *testing.B) {
	cron := &PkgMeta{Binarypkg: "cron", Suite: "testing"}
	systemdCron := &PkgMeta{
		Binarypkg: "manpages-fr-systemd",
		Suite:     "testing",
		Replaces:  []string{"manpages-fr-extra", "manpages-fr", "cron"},
	}
	for b.Loop() {
		boolSink = systemdCron.SameBinary(cron)
	}
}

var (
	sink     string
	boolSink bool
)
