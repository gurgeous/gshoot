[![test](https://github.com/gurgeous/gshoot/actions/workflows/ci.yml/badge.svg)](https://github.com/gurgeous/gshoot/actions/workflows/ci.yml)

<img src="./logo.svg" width="60%">

# gshoot

`gshoot` is a CLI to magically import and export CSVs from Google Sheets. It has a few carefully chosen features along those lines.

For example, if I'm analyzing my local zoo I might run `gshoot up Zoo zoo.csv --numeric --layout --filter` to create a nice-looking Google Sheet. If I add more rows to `zoo.csv`, I can run `gshoot up Zoo zoo.csv --refill` to add the new data without messing up the Google Sheet. For some projects I do this dozens of times a day. Some examples:

```sh
# download the Google Sheets spreadsheet file "Zoo" to /tmp/zoo.csv
$ gshoot down Zoo -o /tmp/zoo.csv

# upload zoo.csv back into the Zoo Google Sheets file
$ gshoot up Zoo /tmp/zoo.csv
```

That's it, that's the whole thing. `gshoot` wraps [gogcli](https://github.com/openclaw/gogcli), which handles Google authentication and API access.

It looks pretty great if you pipe the download into [tennis](https://github.com/gurgeous/tennis)

<img width="60%" src="https://github.com/user-attachments/assets/c0ea68d2-72b5-45d2-96da-259c3e6719e6" />

## Installation

On macOS or Linux with Homebrew:

```sh
brew install openclaw/tap/gogcli
brew install gurgeous/tap/gshoot
```

Other gshoot builds are on the
[GitHub releases page](https://github.com/gurgeous/gshoot/releases/latest).
Install `gog` 0.37.0 or newer separately when not using Homebrew.

## Authentication

Authenticate with gogcli, then choose its default account. gshoot inherits the
normal gog configuration and environment.

```sh
gog auth credentials ~/Downloads/client_secret_*.json
gog auth add you@example.com --services drive,sheets
gog auth alias set default you@example.com
```

See the [gogcli quickstart](https://github.com/openclaw/gogcli/blob/main/docs/quickstart.md)
for complete setup instructions.

### Important Features

- download a CSV from a Google Sheets file (and maybe a specific sheet)
- upload a CSV into a Google Sheets file (and maybe replace/merge into an existing sheet)
- join CSV columns into an existing sheet without overwriting its data
- `up --replace` mode to overwrite an existing sheet
- `up --refill` mode to merge data into an existing sheet, leaving other columns untouched
- `up` has lots of little helpers to make life easier like `--filter`, `--layout`, `--numeric` and `--open`

### Options

```
$ gshoot --help

Magically upload/download CSVs from Google Sheets.

Commands:
  down           Download a Google Sheet as CSV.
  join           Join a CSV into an existing Google Sheet.
  up             Upload a CSV to Google Sheets.
  list           List your Google Sheets.
  peek           List sheets in a spreadsheet.
  wipe           Wipe/delete all data from a spreadsheet.
```

### Up, Up, Up

There are three different modes for `gshoot up`.

1. **safe** (default). A new sheet will be added to the file. The sheet name comes from the CSV filename, so `Something Awesome.tsv` becomes `Something Awesome`. When run repeatedly, you will see new sheets like `Something Awesome`, `Something Awesome_2`, `Something Awesome_3`...
2. **`--replace`** will find/create the sheet in that file and overwrite it with the CSV data.
3. **`--refill`** will merge the CSV data into an existing sheet. It will update old rows and add new ones as necessary. gshoot won't really mess with unknown columns, but it will _extend_ formulas and formatting into new rows.

When using `up`, gshoot will find or create the spreadsheet file as necessary. The target sheet name comes from the CSV filename, which you can override with `--sheet`. I almost always use `--filter`, `--layout`, `--numeric` and `--open` too.

Spreadsheet arguments accept an exact Drive name, spreadsheet ID, or Google Sheets URL.

gog currently cannot bound raw grid-data reads by sheet or range, so `--refill`
and `--layout` fetch grid data for the whole spreadsheet.

### Down, Down, Down

`gshoot down` is much simpler. By default it downloads the first sheet, but you can override with `--sheet`.

### Join

`gshoot join` mixes CSV data into an existing sheet using a shared key column:

```sh
gshoot join Zoo prices.csv --key asin
```

Existing values are never overwritten. CSV-only columns are appended, while a
column present in both inputs is inserted beside the existing column with a `2`
suffix (`price` becomes `price2`). A first column named `join` labels each row as
`left`, `right`, or `match`.

Use `--sheet` to select a sheet, `--columns price,rank` to limit CSV columns, and
`--force` to skip confirmation. gshoot always previews the join and creates a
timestamped backup of the spreadsheet before writing. If the sheet has a filter,
its range is expanded, but existing filtering, sorting, and hidden-row criteria are
lost when the filter is reapplied.

### Other Commands

These are a few other commands for convenience:

- `list` - list recently edited spreadsheet files
- `peek` - list the sheets in a spreadsheet file
- `wipe` - delete all sheets from a spreadsheet file

## Changelog

### 0.2.0 (unreleased)

- Use [gogcli](https://gogcli.sh/) as client instead of direct api

### 0.1.0 (Jun 2026)

- Initial release.

## Future Work

- `gshoot append` to append a csv to a sheet (cols must be identical)
- `ghoost hyperlink plaintext_col link_col`, replace plaintext_col with `=hyperlink(plain, link)`. handle blanks, fail fast on bad links too

## Potential gogcli improvements

These additions are ordered by priority for gshoot:

- `gog sheets batch-request <id> --requests-json @-` for atomic
  `spreadsheets.batchUpdate` requests. gshoot currently starts a metadata command
  and a mutation command for every ordinary operation, so 20 format/copy operations
  mean 40 gog invocations. A batch would reduce those to one and, more importantly,
  prevent a failure from leaving the sheet half-updated.

- `gog sheets raw --range … --fields …` for bounded grid-data reads. Refill and
  layout each need selected grid data from one sheet, but gshoot must currently run
  `sheets raw --include-grid-data` for the entire spreadsheet. This is still one
  request, but a workbook with ten similarly sized sheets can return roughly ten
  times the needed cell data; large unrelated sheets can make the request fail.

- `gog sheets duplicate-tab <id> <sheet> <new-name> [--index N]` for a
  `DuplicateSheetRequest`. This would enable complete in-file sheet snapshots;
  `copy-paste` only copies cell ranges and misses sheet-level state. `join` currently
  uses `gog sheets copy` to back up the entire spreadsheet instead.

- `gog sheets resize-columns --auto --padding N --max-width N` for bounded layout
  in one command. For 20 columns, gshoot currently starts 43 gog commands: two to
  autosize, one raw grid-data read, then a metadata read and resize for each column.
  gog could do the autosize/read/bounded-resize workflow internally in one invocation
  and roughly three API requests.

- `gog drive ls --order-by modifiedByMeTime desc`. `gshoot list` currently makes one
  limited `drive ls` request and already selects only the fields it displays, but
  gog cannot ask Drive to sort first. Fetching everything and sorting locally is the
  only reliable workaround; without it, the requested limit may not contain the
  most recently edited files.

- `gog sheets clear --all-cell-data` to clear values, formats, notes, and validation
  together. One `--replace` clear currently starts five gog commands: one metadata
  read, then separate value, format, validation, and note clears. A single clear
  operation would be faster and could not leave some kinds of old cell state behind.

- `gog sheets resize-grid <id> <sheet> --rows N --columns N` for exact grid
  dimensions. Resizing an existing sheet currently takes a metadata read plus as
  many as two insert/delete commands, one per dimension. This would turn the two
  mutations into one; the latency saving is modest, but rows and columns would no
  longer be left at different sizes after a partial failure.

- `gog sheets paste-data` with stdin, delimiter, and paste-type options. This would
  not save an API request: gshoot already pastes all values in one call. It would
  avoid marshaling the complete in-memory CSV/TSV into a second JSON payload, which
  mainly reduces memory and encoding overhead for large uploads.
