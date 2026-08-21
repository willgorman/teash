package app

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// Column name constants for the static (non-label) server fields.
const (
	ColHostname = "hostname"
	ColIP       = "ip"
	ColOS       = "os"
)

// ColumnFilter restricts servers to those whose value for Column matches at
// least one entry in Values (OR within a column). A ColumnFilter with no
// Values is ignored by FilterServers.
type ColumnFilter struct {
	Column string
	Values []string
}

func columnValue(sv ServerView, column string) string {
	switch column {
	case ColHostname:
		return sv.Hostname
	case ColIP:
		return sv.Addr
	case ColOS:
		return sv.OS
	default:
		return sv.AllLabels[column]
	}
}

// FilterServers applies colFilters to servers with AND-across-columns,
// OR-within-a-column semantics: a server is kept if, for every ColumnFilter
// with at least one value, its column value contains (case-insensitively)
// at least one of those values.
func FilterServers(servers []ServerView, colFilters []ColumnFilter) []ServerView {
	var result []ServerView
	for _, sv := range servers {
		if serverMatchesAllFilters(sv, colFilters) {
			result = append(result, sv)
		}
	}
	return result
}

func serverMatchesAllFilters(sv ServerView, colFilters []ColumnFilter) bool {
	for _, cf := range colFilters {
		if len(cf.Values) == 0 {
			continue
		}
		fieldVal := strings.ToLower(columnValue(sv, cf.Column))
		matchesAny := false
		for _, v := range cf.Values {
			if strings.Contains(fieldVal, strings.ToLower(v)) {
				matchesAny = true
				break
			}
		}
		if !matchesAny {
			return false
		}
	}
	return true
}

// DistinctColumnValues returns the sorted, deduped set of non-empty values
// present for the given column across servers.
func DistinctColumnValues(servers []ServerView, column string) []string {
	seen := make(map[string]bool)
	for _, sv := range servers {
		v := columnValue(sv, column)
		if v != "" {
			seen[v] = true
		}
	}
	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}

// FilterServersByText fuzzy-matches a whitespace-separated query against all
// columns (hostname, IP, OS, labels). Each word in the query must fuzzy-match
// at least one column (AND across words, OR across columns), so e.g. "rocky dev"
// matches a server with OS "Rocky Linux 9" and label "environment: dev".
func FilterServersByText(servers []ServerView, query string) []ServerView {
	words := strings.Fields(query)
	var result []ServerView
	for _, sv := range servers {
		fields := make([]string, 0, 3+len(sv.AllLabels))
		fields = append(fields, sv.Hostname, sv.Addr, sv.OS)
		for _, v := range sv.AllLabels {
			fields = append(fields, v)
		}

		matchesAllWords := true
		for _, word := range words {
			if len(fuzzy.Find(word, fields)) == 0 {
				matchesAllWords = false
				break
			}
		}
		if matchesAllWords {
			result = append(result, sv)
		}
	}
	return result
}
