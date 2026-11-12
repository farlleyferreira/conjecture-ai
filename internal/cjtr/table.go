package cjtr

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

// Table is one table-lookup concept's loaded CSV data: a header row naming
// its columns, and the data rows beneath it.
type Table struct {
	Header []string
	Rows   [][]string
}

// Tables maps a table concept's name to its loaded [Table].
type Tables map[string]*Table

// LoadTable reads path as a CSV file: its first row is the header, every
// row after that is data. Every row, including the header, must have the
// same number of fields (encoding/csv's default behavior) - a short or
// long row is a malformed table, not something to silently pad or trim.
func LoadTable(path string) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cjtr: loading table %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("cjtr: loading table %q: %w", path, err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("cjtr: table %q has no header row", path)
	}

	return &Table{Header: records[0], Rows: records[1:]}, nil
}

// LoadTables loads every table concept's CSV file, resolving each one's
// declared path against baseDir.
func LoadTables(concepts map[string]Concept, baseDir string) (Tables, error) {
	tables := make(Tables)
	for name, c := range concepts {
		if c.Kind != TableConcept {
			continue
		}
		table, err := LoadTable(filepath.Join(baseDir, c.TablePath))
		if err != nil {
			return nil, fmt.Errorf("cjtr: concept %q: %w", name, err)
		}
		tables[name] = table
	}
	return tables, nil
}

// Lookup finds the single row whose by-key columns match keys (a by-key
// name -> its resolved value, decimal-aware per keyConcepts' declared
// type) and returns the value held in the table's one remaining column.
//
// The specification's worked example never names that remaining column,
// so this is the only reading that does not invent a naming convention it
// doesn't declare: a table-lookup concept's "by" keys name every column
// except one, and that one is its value.
func (t *Table) Lookup(keys map[string]string, keyConcepts map[string]Concept) (string, error) {
	valueCol := -1
	keyCols := make(map[string]int, len(keys))
	for i, col := range t.Header {
		if _, isKey := keys[col]; isKey {
			keyCols[col] = i
			continue
		}
		if valueCol != -1 {
			return "", fmt.Errorf("cjtr: table has more than one column besides its by-keys (%q and %q); which one is the value is not declared",
				t.Header[valueCol], col)
		}
		valueCol = i
	}
	if valueCol == -1 {
		return "", fmt.Errorf("cjtr: table has no column left for its value besides its by-keys")
	}

	matches, err := t.matchingRows(keys, keyCols, keyConcepts)
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("cjtr: table has no row matching %v", keys)
	case 1:
		return t.Rows[matches[0]][valueCol], nil
	default:
		return "", fmt.Errorf("cjtr: table has more than one row matching %v", keys)
	}
}

// MatchCount reports how many of t's own rows match keys (a by-key
// name -> its resolved value, the same shape [Table.Lookup] itself
// takes) - 0, 1, or more than 1. It exists so a caller that only cares
// whether a combination of by-key values has any row at all (omission-
// detection's own Type C, "a case missing from a decision table") can
// ask that without also needing a value column to exist, which
// [Lookup] itself requires but this does not.
func (t *Table) MatchCount(keys map[string]string, keyConcepts map[string]Concept) (int, error) {
	keyCols := make(map[string]int, len(keys))
	for i, col := range t.Header {
		if _, isKey := keys[col]; isKey {
			keyCols[col] = i
		}
	}
	matches, err := t.matchingRows(keys, keyCols, keyConcepts)
	if err != nil {
		return 0, err
	}
	return len(matches), nil
}

// matchingRows returns the row indices matching keys, resolved
// against keyCols (keys's own column name -> header index) - the
// shared row-scan both [Lookup] and [MatchCount] run, differing only
// in what they do with the result.
func (t *Table) matchingRows(keys map[string]string, keyCols map[string]int, keyConcepts map[string]Concept) ([]int, error) {
	for name := range keys {
		if _, ok := keyCols[name]; !ok {
			return nil, fmt.Errorf("cjtr: table has no column named %q for its by-key", name)
		}
	}

	var matches []int
	for i, row := range t.Rows {
		rowMatches := true
		for name, col := range keyCols {
			eq, err := valuesEqual(keyConcepts[name], row[col], keys[name])
			if err != nil {
				return nil, fmt.Errorf("cjtr: comparing by-key %q: %w", name, err)
			}
			if !eq {
				rowMatches = false
				break
			}
		}
		if rowMatches {
			matches = append(matches, i)
		}
	}
	return matches, nil
}
