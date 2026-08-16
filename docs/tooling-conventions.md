# Tooling Conventions

This document owns three conventions for how commands and edits are run in
this repo: locating Go module source, waiting on long-running processes, and
general shell/editing hygiene.

## Locating Go module source

Never sweep the filesystem to locate a Go dependency's source. Ask the
toolchain instead:

```sh
go list -m -f '{{.Dir}}' <module>
```

This prints the directory in ~0.02s, whether the module resolves to the
module cache or to a local `replace`. The same applies to `go doc <pkg>` for
a symbol and `go list -m all` for the version set.

`find /` takes ~2 minutes on WSL2. One task-227 session burned 6 minutes
across five whole-filesystem sweeps hunting for `atlas-rest`, which `go.mod`
had `replace`d to `libs/atlas-rest` inside the worktree the agents were
already working in. Guessing at module-cache case-escaping (`!chronicle20`)
is the tell that you should have asked `go list` instead. `find` is for
paths you own, rooted at a directory you name — never at `/`.

## Waiting on processes

Never spend inference turns waiting for a process. Launch it once with a
bound — `run_in_background: true`, or `Monitor` with an until-loop — and do
something else or hand back.

Repeated `sleep` / `ps aux | grep` / `echo waiting` / `for i in $(seq …); do
sleep` calls are the anti-pattern: each one re-reads the whole context to
learn nothing, and they cluster late in a session where that is most
expensive. If the process exceeds its bound, kill it and fall back; do not
keep polling. When a tool has a known hang mode, the fallback belongs in
that tool's agent doc, not in a longer wait.

## Shell and editing conventions

Prefer portable POSIX shell; avoid zsh/direnv-specific constructs and batch
patch loops that can produce garbled or unapplied output. For a multi-file
edit, prefer per-file Edit/Write over a shell patch loop.

Quote glob arguments in shell tool calls — `--include='*.go'`, not
`--include=*.go` — zsh expands an unquoted glob before `grep` sees it,
producing `no matches found` and a wasted retry.

Preserve line endings when editing — do not normalize CRLF→LF as a side
effect; it inflates diffs with spurious changes.

Always use repo-relative paths or placeholders in committed files; never
literal home or absolute paths like `/Users/<name>/...` or
`/home/<name>/...` — a committed absolute path is not reproducible on
another machine.
