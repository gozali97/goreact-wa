package logsvc

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Query describes a log read request.
type Query struct {
	Day     string // YYYY-MM-DD (defaults to today if empty)
	Search  string // case-insensitive substring over message/source/fields
	Status  string // filter by status (success/failed/issue/info); empty = all
	Level   string // filter by level; empty = all
	Page    int    // 1-based
	PerPage int    // page size
}

// Result is a paginated query response.
type Result struct {
	Day     string  `json:"day"`
	Total   int     `json:"total"` // matched entries
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
	Pages   int     `json:"pages"`
	Entries []Entry `json:"entries"`
}

// Days returns the available log dates (YYYY-MM-DD), newest first.
func (s *Service) Days() ([]string, error) {
	if s == nil {
		return nil, nil
	}
	matches, err := filepath.Glob(filepath.Join(s.dir, "wa-proxy-*.log"))
	if err != nil {
		return nil, err
	}
	days := make([]string, 0, len(matches))
	for _, m := range matches {
		base := filepath.Base(m)
		day := strings.TrimSuffix(strings.TrimPrefix(base, "wa-proxy-"), ".log")
		if len(day) == 10 {
			days = append(days, day)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(days)))
	return days, nil
}

// readAll reads and parses all entries for a day (newest first).
func (s *Service) readAll(day string) ([]Entry, error) {
	f, err := os.Open(s.fileName(day))
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	// Allow long lines (large field payloads).
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	// Newest first.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, scanner.Err()
}

// matches applies the search/status/level filters to an entry.
func (q Query) matches(e Entry) bool {
	if q.Status != "" && e.Status != q.Status {
		return false
	}
	if q.Level != "" && e.Level != q.Level {
		return false
	}
	if q.Search != "" {
		needle := strings.ToLower(q.Search)
		hay := strings.ToLower(e.Message + " " + e.Source + " " + e.Status + " " + e.Level)
		if !strings.Contains(hay, needle) {
			// Also search field values.
			found := false
			for k, v := range e.Fields {
				if strings.Contains(strings.ToLower(k), needle) ||
					strings.Contains(strings.ToLower(toStr(v)), needle) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// QueryEntries reads, filters and paginates entries for the requested day.
func (s *Service) QueryEntries(q Query) (*Result, error) {
	if q.PerPage <= 0 {
		q.PerPage = 50
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	day := q.Day
	if day == "" {
		day = dateKey(timeNow())
	}

	all, err := s.readAll(day)
	if err != nil {
		return nil, err
	}

	filtered := make([]Entry, 0, len(all))
	for _, e := range all {
		if q.matches(e) {
			filtered = append(filtered, e)
		}
	}

	total := len(filtered)
	pages := (total + q.PerPage - 1) / q.PerPage
	if pages == 0 {
		pages = 1
	}
	start := (q.Page - 1) * q.PerPage
	if start > total {
		start = total
	}
	end := start + q.PerPage
	if end > total {
		end = total
	}

	return &Result{
		Day:     day,
		Total:   total,
		Page:    q.Page,
		PerPage: q.PerPage,
		Pages:   pages,
		Entries: filtered[start:end],
	}, nil
}

// FilePath returns the absolute path to a day's log file (for download).
func (s *Service) FilePath(day string) string {
	if day == "" {
		day = dateKey(timeNow())
	}
	return s.fileName(day)
}

func toStr(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
