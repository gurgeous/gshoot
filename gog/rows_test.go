package gog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRowsValidateHeaders(t *testing.T) {
	assert.EqualError(t, (Rows{{"", ""}}).ValidateHeaders("csv"), "csv has no headers")
	assert.EqualError(t, (Rows{{"id", "name", "id"}}).ValidateHeaders("csv"), "csv has duplicate headers: id")
	assert.NoError(t, (Rows{{"id", "name"}}).ValidateHeaders("csv"))
}

func TestRowsIndexes(t *testing.T) {
	rows := Rows{{"id", "name"}, {"1", "Ada"}, {"", "Blank"}, {"2", "Bob"}}

	assert.Equal(t, map[string]int{"id": 0, "name": 1}, rows.ColumnIndexes())
	indexes, err := rows.UniqueRowIndexes(0, "sheet")
	assert.NoError(t, err)
	assert.Equal(t, map[string]int{"1": 1, "2": 3}, indexes)

	_, err = (Rows{{"id"}, {"1"}, {"1"}}).UniqueRowIndexes(0, "sheet")
	assert.EqualError(t, err, `sheet has duplicate key "1"`)
}
