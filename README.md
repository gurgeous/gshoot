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
- `up --replace` mode to overwrite an existing sheet
- `up --refill` mode to merge data into an existing sheet, leaving other columns untouched
- `up` has lots of little helpers to make life easier like `--filter`, `--layout`, `--numeric` and `--open`

### Options

```
$ gshoot --help

Magically upload/download CSVs from Google Sheets.

Commands:
  down           Download a Google Sheet as CSV.
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

### Other Commands

These are a few other commands for convenience:

- `list` - list recently edited spreadsheet files
- `peek` - list the sheets in a spreadsheet file
- `wipe` - delete all sheets from a spreadsheet file

## Changelog

### 0.1.0 (unreleased)

- Initial release.

## Potential gogcli improvements

These additions would make gshoot faster and simpler:

- `gog sheets batch-request <id> --requests-json @-` for atomic `spreadsheets.batchUpdate` requests.
- `gog sheets raw --range … --fields …` for bounded grid-data reads.
- `gog sheets resize-grid <id> <sheet> --rows N --columns N` for exact grid dimensions.
- `gog sheets resize-columns --auto --padding N --max-width N` for bounded layout in one command.
- `gog sheets paste-data` with stdin, delimiter, and paste-type options.
- `gog sheets clear --all-cell-data` to clear values, formats, notes, and validation together.
- `gog drive ls/search --order-by` and `gog drive search --fields` for `modifiedByMeTime` workflows.
