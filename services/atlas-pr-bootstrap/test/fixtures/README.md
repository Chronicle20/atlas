# rpk JSON-output fixtures

Pinned to `ARG RPK_VERSION=24.3.1` in `../../Dockerfile`. Bumping
that version invalidates these files; regenerate against the new
rpk binary and re-run `bats services/atlas-pr-bootstrap/test/`.

## Regenerate

Run against any reachable Kafka broker (e.g. the cluster's
`kafka.home:9093`):

```
rpk topic list -X brokers=<broker> --format json \
  > services/atlas-pr-bootstrap/test/fixtures/rpk-topic-list.json
rpk group list -X brokers=<broker> --format json \
  > services/atlas-pr-bootstrap/test/fixtures/rpk-group-list.json
```

After regenerating, edit the files to keep the test scenarios
intact:

- One topic name ending in `-a1b2` plus one not ending in
  `-a1b2` (cleanup-side suffix test).
- One group name ending in `[a1b2]` containing spaces, one
  ending in `[a1b2]` without spaces, one ending in `[other]`
  (cleanup-side group suffix + spaced-name test).

`a1b2` is a literal `ATLAS_ENV` value the bats tests use directly
(via `make_stubs`); other tests compute their own env hash from
`PR_NUMBER` and sed-substitute fixture copies — see
`cleanup_test.bats::make_stubs`.
