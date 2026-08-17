package recode

import (
	"bytes"
	"io"
	"testing"
)

// Every non-UTF-8 manpage is streamed through recode.Reader before it is
// handed to mandoc, so the decoder throughput directly influences the
// extraction stage.
func BenchmarkReader(b *testing.B) {
	src, err := readGzipped("../../testdata/kterm.1.ja.gz")
	if err != nil {
		b.Fatal(err)
	}

	// "ja" uses EUC-JP, "ru" uses KOI8-R and "en" falls back to the
	// default ISO-8859-1 encoding, which exercises the three kinds of
	// decoders debiman relies on.
	for _, lang := range []string{"ja", "ru", "en"} {
		b.Run(lang, func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			for b.Loop() {
				if _, err := io.Copy(io.Discard, Reader(bytes.NewReader(src), lang)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
