package cli

import "testing"

func TestSpreadsheetNotFoundError(t *testing.T) {
	t.Parallel()

	err := spreadsheetNotFoundError("Budget")
	if got, want := err.Error(), "spreadsheet not found: Budget"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestSheetNotFoundError(t *testing.T) {
	t.Parallel()

	err := sheetNotFoundError("Budget", "Q1")
	if got, want := err.Error(), "sheet not found in Budget: Q1"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNoSheetsError(t *testing.T) {
	t.Parallel()

	err := noSheetsError("Budget")
	if got, want := err.Error(), "spreadsheet has no sheets: Budget"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
