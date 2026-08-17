# debiman performance

The tables below describe full debiman runs. In both cases, a Debian mirror was
available via a Gigabit ethernet connection.

## Modern machine

Intel® Core™ i7-6700K (8 x 4.0 GHz), 32 GB DDR4-RAM, Intel SSDSC2BP48

&nbsp;                 | Debian unstable | all Debian suites
-----------------------|-----------------|-----------------------
contents parsing       | <6s             | <25s
package parsing        | <2s             | <20s
xref preparation       | <3s             | <15s
stat                   | <2s             | <4s
**total incremental**  | **<13s**        | **<65s**
full man extraction    | 5m              | 20m
full man rendering     | <4m             | 12m
**total from scratch** | **<10m**        | **34m**

## Dated machine

AMD Opteron™ 23xx (2 x 2.2 GHz), 2 GB RAM, TODO spinning disk

&nbsp;                 | Debian unstable | all Debian suites
-----------------------|-----------------|-----------------------
contents parsing       | <70s            | <167s
package parsing        | <10s            | <35s
xref preparation       | (not measured)  | <80s
stat                   | (not measured)  | <60s
**total incremental**  | **<140s**       | **<10m** (TODO)
full man extraction    | TODO            | TODO
full man rendering     | TODO            | 2h
**total from scratch** | TODO            | TODO

## Continuous benchmarking

The hot code paths of debiman are covered by Go benchmarks, which live next to
the code they measure in files named `*_bench_test.go`:

* `internal/tag`: locale to BCP-47 language tag conversion
* `internal/manpage`: parsing manpage paths and serving paths
* `internal/recode`: recoding non-UTF-8 manpages
* `internal/sitemap`: writing sitemaps
* `internal/redirect`: loading the index and resolving redirects (debiman-auxserver)
* `cmd/debiman`: Contents parsing, re-using rendered manpages and HTML rendering

To run them locally:

```
go test -bench=. ./...
```

They also run on every push and pull request and are tracked over time by
[CodSpeed](https://app.codspeed.io/maksimtech/debiman).
