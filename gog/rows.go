package gog

import (
	"fmt"
	"sort"
	"strings"
)

//
// Structural validation and indexing for rectangular spreadsheet rows.
//

// ValidateHeaders rejects missing or duplicate named headers.
func (r Rows) ValidateHeaders(label string) error {
	if len(r) == 0 {
		return fmt.Errorf("%s has no headers", label)
	}

	counts := map[string]int{}
	for _, header := range r[0] {
		if header != "" {
			counts[header]++
		}
	}
	if len(counts) == 0 {
		return fmt.Errorf("%s has no headers", label)
	}

	duplicates := []string{}
	for header, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, header)
		}
	}
	if len(duplicates) == 0 {
		return nil
	}
	sort.Strings(duplicates)
	return fmt.Errorf("%s has duplicate headers: %s", label, strings.Join(duplicates, ", "))
}

// ColumnIndexes maps each header to its column index.
func (r Rows) ColumnIndexes() map[string]int {
	indexes := make(map[string]int, len(r[0]))
	for i, header := range r[0] {
		indexes[header] = i
	}
	return indexes
}

// ColumnBlank reports whether every data cell in a column is empty.
func (r Rows) ColumnBlank(column int) bool {
	for _, row := range r[1:] {
		if row[column] != "" {
			return false
		}
	}
	return true
}

// UniqueRowIndexes indexes nonblank keys and rejects duplicates.
func (r Rows) UniqueRowIndexes(column int, label string) (map[string]int, error) {
	indexes := map[string]int{}
	for i := 1; i < len(r); i++ {
		key := r[i][column]
		if key == "" {
			continue
		}
		if _, ok := indexes[key]; ok {
			return nil, fmt.Errorf("%s has duplicate key %s=%s", label, r[0][column], key)
		}
		indexes[key] = i
	}
	return indexes, nil
}
