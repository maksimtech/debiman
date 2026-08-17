package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"
)

// benchContents generates a Contents-<arch> file body which resembles the
// real ones: most lines do not refer to a manpage and have to be skipped,
// and some files are owned by more than one binary package.
func benchContents(pkgs int) []byte {
	var buf bytes.Buffer
	for i := 0; i < pkgs; i++ {
		fmt.Fprintf(&buf, "usr/bin/binary-%d                                        utils/pkg-%d\n", i, i)
		fmt.Fprintf(&buf, "usr/share/doc/pkg-%d/changelog.Debian.gz                 utils/pkg-%d\n", i, i)
		fmt.Fprintf(&buf, "usr/share/man/man1/binary-%d.1.gz                         utils/pkg-%d\n", i, i)
		fmt.Fprintf(&buf, "usr/share/man/fr/man1/binary-%d.1.gz                      utils/pkg-%d\n", i, i)
		if i%16 == 0 {
			// A manpage owned by multiple binary packages.
			fmt.Fprintf(&buf, "usr/share/man/man5/shared-%d.5.gz                         utils/pkg-%d,utils/pkg-%d,utils/pkg-%d\n", i, i, i+1, i+2)
		}
	}
	return buf.Bytes()
}

// Parsing the Contents files of all architectures of all suites is the
// first stage of a debiman run and takes tens of seconds for all of
// Debian, see PERFORMANCE.md.
func BenchmarkParseContentsEntry(b *testing.B) {
	contents := benchContents(5000)
	b.SetBytes(int64(len(contents)))

	for b.Loop() {
		scanner := bufio.NewScanner(bytes.NewReader(contents))
		scanner.Buffer(nil, 512*1024)
		var entries int
		for {
			e, err := parseContentsEntry(scanner)
			if err != nil {
				if err == io.EOF {
					break
				}
				b.Fatal(err)
			}
			entries += len(e)
		}
		if entries == 0 {
			b.Fatal("parseContentsEntry unexpectedly found no manpages")
		}
	}
}
