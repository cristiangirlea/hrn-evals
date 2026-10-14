---
name: check
description: Run hrn-evals' offline checks (Go build and vet, script syntax, suite lint) before a PR; never runs an eval
---
CI runs these same checks; a PR must pass them. From the repository root, in Git Bash, in
this order, stopping at the first failure:

```bash
go build ./... && go vet ./...
bash -n scripts/run-all.sh
node -e '
const fs=require("fs"),p=require("path");let bad=0;
for(const k of fs.readdirSync("suites"))for(const f of fs.readdirSync(p.join("suites",k)))if(f.endsWith(".json")){
 const n=p.join("suites",k,f);let s;try{s=JSON.parse(fs.readFileSync(n,"utf8"))}catch(e){console.log(n+": "+e.message);bad=1;continue}
 if(s.dir!=="../.."){console.log(n+": dir is not ../..");bad=1}
 for(const t of [s,...(s.tasks||[])])if("retry" in t){console.log(n+": retry is refused in an eval spec");bad=1}
 for(const t of s.tasks||[])if(!/Do not run git\.$/.test(t.prompt)){console.log(n+": "+t.id+": prompt does not end with \"Do not run git.\"");bad=1}
}
process.exit(bad)'
```

Do not run `go test ./...`: `fixtures/buggy` fails on purpose. Do not run `hrn eval` or
`scripts/run-all.sh`; they start real agent runs, and only the owner starts those.

Report one line per step (pass, or the failure's message). If a suite's gates changed, say that
they still need checking both ways (README, "Adding a suite", step 4) and a run on each variant
by the owner. Never push to `main`. This skill does not merge: merging is through the PR, under
AGENTS.md's conditions.
