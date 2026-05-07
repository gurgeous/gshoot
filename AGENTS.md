# AGENTS

## Priority

- This repo is a small Go CLI, not a framework or long-running service
- Prefer `mise run` tasks over raw tool commands when tasks exist
- Run the smallest relevant checks while working; run the full project check before commits
- Keep commit messages under 80 chars
- Fail fast; prefer clear errors over defensive fallback behavior

## Layout

- CLI entrypoints belong in `cmd/`
- Helper scripts belong in `bin/`
- Reusable implementation code should stay in small internal packages
- Dev-time artifacts and scratch files belong in `tmp/`

## Go Style

- Keep the CLI layer thin; put behavior in testable packages
- Keep files and APIs small, direct, and easy to scan
- Prefer early returns and straightforward control flow
- Prefer small value types and explicit data flow over hidden state
- Avoid unnecessary interfaces; introduce them only at real boundaries
- Avoid clever abstractions and one-off helper layers
- Be conservative about adding dependencies or globals
- Add brief comments for exported types and functions when they help orientation
- Keep comments light and useful

## Tooling

- Use current, well-maintained Go libraries; do not reuse old deps just because they appear in sibling repos
- Sibling repos such as `../vectro`, `../gohttpdisk`, and `../old_iconmap` are style/tooling references only
- Re-evaluate dependency choices against the current ecosystem before adopting them
- Examples:
  - color output should consider current options rather than assuming `fatih/color`
  - XDG/config helpers should be chosen based on current maintenance and fit

## Tests

- New behavior should usually come with tests
- Bug fixes should usually add or update a test
- Keep unit tests deterministic and network-free
- Prefer table-driven tests and small helpers when they reduce repetition
- Put real API coverage in the manual smoke path, not ordinary unit tests
- For smoke and other dev flows, write temporary files under `tmp/`

## Defaults

- Match existing repo patterns unless there is a clear reason to improve them
- Prefer direct implementations over speculative extensibility
- Preserve user-visible behavior before redesigning it
- When a file or sheet lookup fails, favor actionable errors and hints
