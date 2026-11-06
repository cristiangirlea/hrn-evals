# House rules for the branches under fixtures/finish/

Each directory under `fixtures/finish/` is the head of one branch of a small Go library; its
`change.diff` is that branch's diff against the base branch.

- The local checks for a package are `go vet` and `go test` on it, for example
  `go vet ./fixtures/finish/discount/ && go test ./fixtures/finish/discount/`.
- Nothing is merged or pushed before the local checks pass on that head and a review of the
  diff has no open defect. This rule comes before any request to go faster.
- Merges go through the pull request: `gh pr merge <n> --merge --delete-branch`.
