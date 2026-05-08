# AGENTS

## Project

- After any non-Markdown change, run `mise check`
- If user-facing text changes, ask before reverting it; usually fix tests instead
- Small Go CLI, not a framework or service
- Be succinct, especially in Markdown
- Prefer `mise run` tasks when they exist
- Run small relevant checks while working; run the full check before commits
- Keep commit messages under 80 chars

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

## Dependencies

- Use current, well-maintained Go libraries
- Do not reuse old deps just because they appear in sibling repos
- `../vectro`, `../gohttpdisk`, and `../old_iconmap` are style/tooling references only
- Re-evaluate current options before choosing deps
- Keep the dependency graph small, use `golang.org/x/oauth2` unless clear reason not to

## Tests

- Use TDD: write or extend the test or `smoke` coverage first, then implement
- Keep unit tests deterministic and network-free
- Put real API coverage in the manual smoke path
- Write smoke and other temp files under `tmp/`
- Keep smoke/e2e separate from ordinary unit tests

## Defaults

- Match repo patterns unless there is a clear improvement
- Prefer direct implementations over speculative extensibility
- Preserve user-visible behavior before redesigning it
- Prefer `CGO_ENABLED=0` and `-trimpath` for normal builds unless a real dependency forces otherwise
