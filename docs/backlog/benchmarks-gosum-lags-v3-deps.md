---
worth: later
where: benchmarks/go.sum
added: 2026-11-09
---
# benchmarks go.sum lags v3's dependency bump

The benchmarks module builds against the local v3 module (`replace github.com/go-pkgz/expirable-cache/v3 => ../v3`),
but its go.sum still lists testify v1.12.0 and gopkg.in/yaml.v3 v3.0.1. PR #29 bumped v3 to testify v1.12.1,
whose YAML dependency is go.yaml.in/yaml/v3 v3.0.5, and did not touch benchmarks/. `go mod tidy -diff` in
benchmarks/ was clean before that merge and now wants an 8-line swap.

Nothing breaks: testify is only in v3's test dependencies, so `go vet -mod=readonly ./...` still passes in
benchmarks/, and no CI job builds that module. The next `go mod tidy` there will pick up an unrelated go.sum
change. Fix: `go mod tidy` in benchmarks/, ideally the next time that module is edited. The #26 bump tidied
benchmarks together with v3.
