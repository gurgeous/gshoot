package cli

import (
	"strings"
	"testing"
)

func TestSpreadsheetNotFoundError(t *testing.T) {
	t.Parallel()

	err := spreadsheetNotFoundError("Budget")
	if !strings.Contains(err.Error(), "hint: run `gshoot list`") {
		t.Fatalf("error = %q, want list hint", err.Error())
	}
}

func TestSheetNotFoundError(t *testing.T) {
	t.Parallel()

	err := sheetNotFoundError("Budget", "Q1")
	if !strings.Contains(err.Error(), "hint: run `gshoot list`") {
		t.Fatalf("error = %q, want list hint", err.Error())
	}
}

func TestNoSheetsError(t *testing.T) {
	t.Parallel()

	err := noSheetsError("Budget")
	if !strings.Contains(err.Error(), "hint: run `gshoot list`") {
		t.Fatalf("error = %q, want list hint", err.Error())
	}
}
