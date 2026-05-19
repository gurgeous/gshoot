## Important

- Avoid overly specific tests; assert only meaningful behavior
- Over-testing wastes time/money
- In `httptest` flows, only assert important behavior (not every detail)

## Project

- After any non-Markdown change, run `just check`
- Use `just format` instead of `gofmt` directly
- If user-facing text changes, ask before reverting it; usually fix tests instead
- Use `mv` for file moves/renames; use patches for content edits
- When creating a PR, write a succinct title/body from the diff vs `main`; do not use auto-filled wip text
- Small Go CLI, not a framework or service
- Be succinct, especially in Markdown
- Prefer `just` tasks when they exist
- Run small relevant checks while working; run the full check before commits
- Keep commit/PR messages under 80 chars

## Layout

- CLI entrypoints in `cmd/`
- Helper scripts in `bin/`
- Reusable code in small internal packages
- Dev-time artifacts in `tmp/`

## Style

- Keep the CLI thin and behavior testable
- Prefer small, direct code with early returns and explicit data flow
- Avoid unnecessary interfaces, clever abstractions, extra globals, and one-off helper layers
- Keep comments brief and useful
- Fail fast; prefer clear errors and actionable hints

## Design

- Put behavior on the object that owns the data
- Avoid simple single-use methods, especially one-line wrappers
- Avoid defensive code for impossible or unsupported cases
- Keep state and behavior together
- Avoid threading owned state through helper params
- Avoid field bags with many free helpers
- Pass call data directly
- Hide details callers should not know
- Match files and types to real concepts
- Use small internal structs when they clarify ownership
- Remove helpers that only wrap fields
- Before finishing ask who owns this state and what can be hidden

## Dependencies

- Use current, well-maintained Go libraries
- Do not reuse old deps just because they appear in sibling repos
- `../vectro`, `../gohttpdisk`, and `../old_iconmap` are style/tooling references only
- Re-evaluate current options before choosing deps
- Keep the dependency graph small, use `golang.org/x/oauth2` unless clear reason not to

## Tests

- Use TDD: write or extend the test coverage first, then implement
- Keep unit tests deterministic and network-free
- Use `assert.`, not `require.`
- Avoid trivial tests

## Defaults

- Match repo patterns unless there is a clear improvement
- Prefer direct implementations over speculative extensibility
- Preserve user-visible behavior before redesigning it
- Prefer `CGO_ENABLED=0` and `-trimpath` for normal builds unless a real dependency forces otherwise
- Help goes to stdout; progress and errors go to stderr; only animate progress on TTY stderr
