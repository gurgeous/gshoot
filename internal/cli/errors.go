package cli

import "fmt"

func spreadsheetNotFoundError(name string) error {
	return fmt.Errorf("spreadsheet not found: %s\nhint: run `gshoot list`", name)
}

func sheetNotFoundError(spreadsheet, sheet string) error {
	return fmt.Errorf("sheet not found in %s: %s\nhint: run `gshoot list`", spreadsheet, sheet)
}
