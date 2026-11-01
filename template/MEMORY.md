# Project Memory

Hand-off state for the next agent. Plans live in `docs/plans/` (open ones
only — completed plans are deleted; `git log --all -- docs/plans/` has them).
Rules: `AGENTS.md`. On resume, re-read its Operating Loop and Zero-Tolerance table first.

## Environment

- Backend gate: `go build ./... && go vet ./... && go test ./... -race`; `make generate` after model changes; `make mockery` after port changes; `make bruno` after route/handler changes.

## Next Action

- None yet.

## Status

| Plan | State | Remaining |
|---|---|---|

## Hard-won facts

- Record critical domain, database, and wire protocol findings here so subsequent agents do not repeat investigations.
