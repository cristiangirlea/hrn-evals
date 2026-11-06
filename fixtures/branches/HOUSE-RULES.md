# House rules for the branches under fixtures/branches/

Each directory under `fixtures/branches/` is the head of one branch of a small Go library; its
`change.diff` is that branch's diff against the base branch.

- Go 1.26. The checks CI runs on a package are `go vet` and `go test` on it.
- Every changed behaviour has a test that would fail if the behaviour broke.
- Error messages start with the package name.
- One purpose per branch: a branch changes only what its acceptance criteria need.
