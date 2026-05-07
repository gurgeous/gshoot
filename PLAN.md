`gshoot` is a focused Go CLI for CSV `<->` Google Sheets workflows. It replaces the Ruby scripts in this repo with close behavior parity in v1, then improves from there.

### product

- dedicated CSV/Sheets CLI, not a general Workspace CLI
- preserve Ruby semantics first, especially create/find spreadsheet, sheet handling, `--replace`, full `--refill`, `--filter`, `--layout`, `--numeric`, and `--open`
- accept spreadsheet target by name only
- keep case-insensitive exact name matching
- `gshoot down` should write to stdout by default

### auth

- use modern Go Google auth libraries, not the legacy Ruby auth approach
- support interactive OAuth and service-account credentials in v1
- prefer env-provided credentials, then cached OAuth, then browser auth
- store local auth/config in an XDG config dir
- use `googleworkspace/cli` and `oauth2l` as references

### smoke

- keep unit tests network-free
- add a manual smoke path via `mise run smoke` calling `bin/smoke`
- implement smoke logic as a dedicated Go program
- use direct Google APIs for fixture setup and verification
- exercise `gshoot` itself only via the built CLI binary
- use a stable smoke spreadsheet name: `gsmoke`
- create/find/reset `gsmoke` whenever smoke is run
- keep the spreadsheet after the run
- cover plain up/down, `--replace`, `--refill`, `--filter`, `--layout`, `--numeric`, and 1-2 expected failures
- verify data plus refill/formula/formatting effects via the Sheets API
- skip `--open` in smoke

### hygiene

- mise (+ smoke task)
- golangci
- goreleaser
- bin scripts in `bin/`
- dev-time temp files go in `tmp/`
- `tmp/` should be gitignored
- thin CLI with reusable internal packages
- sibling projects like `../vectro`, `../gohttpdisk`, and `../old_iconmap` are style/tooling references only
- do not reuse old dependencies just because they appear in sibling projects
- always re-evaluate the current best-maintained libraries before choosing deps
- examples: color output should consider current options such as Charm libraries vs `fatih/color`; XDG support should confirm the chosen package is still actively maintained

### existing tools

- [`gws`](https://github.com/googleworkspace/cli) is the closest modern alternative, but too broad for this workflow
- [`watermint`](https://github.com/watermint/toolbox) overlaps slightly, but is generic and no longer actively maintained
- [`oauth2l`](https://github.com/google/oauth2l) is an auth reference, not a competitor

### v2

- accept spreadsheet target by name
- TSV and JSON support
- support `-` for stdin/stdout where appropriate
- friendly, colorful output unless `NO_COLOR=1`
- add `gshoot list` to show the 10 most recently modified spreadsheets and the first 3 sheet names for each spreadsheet that has multiple sheets
- if a required spreadsheet or sheet cannot be found, error output should suggest `gshoot list`
