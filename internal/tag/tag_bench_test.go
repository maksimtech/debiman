package tag

import "testing"

// The locales below cover the three code paths of FromLocale: a plain
// language tag, a tag with a codeset which needs to be stripped and a
// tag with a modifier which needs to be mapped to a BCP-47 variant.
var benchLocales = []string{
	"en",
	"de_DE",
	"zh_TW.Big5",
	"ja_JP.EUC-JP",
	"sr@latin",
	"ca_ES.UTF-8@valencia",
}

func BenchmarkFromLocale(b *testing.B) {
	for _, locale := range benchLocales {
		b.Run(locale, func(b *testing.B) {
			for b.Loop() {
				if _, err := FromLocale(locale); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFromLocaleMixed walks the whole set, which is closer to what
// debiman does when it processes the manpages of all Debian suites.
func BenchmarkFromLocaleMixed(b *testing.B) {
	for b.Loop() {
		for _, locale := range benchLocales {
			if _, err := FromLocale(locale); err != nil {
				b.Fatal(err)
			}
		}
	}
}
