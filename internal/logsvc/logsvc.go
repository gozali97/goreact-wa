// Package logsvc provides a small, reusable structured logging service that
// writes one JSON-lines log file per day under a configurable directory. It is
// safe for concurrent use and exposes package-level helpers so any part of the
// app can record events without wiring a logger through.
//
// Each entry records: time, level, status (success/failed/issue/info), source
// (the function/category that emitted it), a human message, and optional
// structured fields. The dashboard log viewer reads these files back.
package logsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Levels.
const (
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// Outcome status values.
const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusIssue   = "issue"
	StatusInfo    = "info"
)

// Entry is a single structured log record (one JSON line in the file).
type Entry struct {
	Time    time.Time              `json:"time"`
	Level   string                 `json:"level"`
	Status  string                 `json:"status"`
	Source  string                 `json:"source"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Service writes entries to per-day files and also mirrors to stdout.
type Service struct {
	dir       string
	mu        sync.Mutex
	day       string // current file's date (YYYY-MM-DD)
	file      *os.File
	toConsole bool
}

var (
	defaultSvc *Service
	once       sync.Once
)

// Init initializes the package-level default service. Safe to call once at
// startup. Subsequent calls are ignored.
func Init(dir string, toConsole bool) error {
	var err error
	once.Do(func() {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			err = fmt.Errorf("create log dir: %w", mkErr)
			return
		}
		defaultSvc = &Service{dir: dir, toConsole: toConsole}
	})
	return err
}

// Default returns the package-level service (may be nil if Init wasn't called).
func Default() *Service { return defaultSvc }

// dateKey returns the YYYY-MM-DD key for a time in local zone.
func dateKey(t time.Time) string { return t.Format("2006-01-02") }

// timeNow is a small indirection over time.Now (kept simple/local).
func timeNow() time.Time { return time.Now() }

// fileName returns the log file path for a given date key.
func (s *Service) fileName(day string) string {
	return filepath.Join(s.dir, "wa-proxy-"+day+".log")
}

// rotateIfNeeded ensures s.file points at today's file (caller holds the lock).
func (s *Service) rotateIfNeeded(now time.Time) error {
	day := dateKey(now)
	if s.file != nil && s.day == day {
		return nil
	}
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
	f, err := os.OpenFile(s.fileName(day), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	s.file = f
	s.day = day
	return nil
}

// write appends an entry to today's file.
func (s *Service) write(e Entry) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.rotateIfNeeded(e.Time); err != nil {
		fmt.Fprintf(os.Stderr, "logsvc: rotate: %v\n", err)
		return
	}
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	_, _ = s.file.Write(append(line, '\n'))

	if s.toConsole {
		fmt.Printf("[%s] %-7s %-28s %s\n",
			e.Time.Format("15:04:05"), e.Status, e.Source, e.Message)
	}
}

// Log records an entry. source is the function/category, status one of the
// Status* constants, fields optional structured data.
func (s *Service) Log(level, status, source, message string, fields map[string]interface{}) {
	s.write(Entry{
		Time:    time.Now(),
		Level:   level,
		Status:  status,
		Source:  source,
		Message: message,
		Fields:  fields,
	})
}

// ─── Package-level convenience helpers ────────────────────

// Success logs a successful operation.
func Success(source, message string, fields map[string]interface{}) {
	defaultSvc.Log(LevelInfo, StatusSuccess, source, message, fields)
}

// Failed logs a failed operation (error-level).
func Failed(source, message string, fields map[string]interface{}) {
	defaultSvc.Log(LevelError, StatusFailed, source, message, fields)
}

// Issue logs a non-fatal problem / warning.
func Issue(source, message string, fields map[string]interface{}) {
	defaultSvc.Log(LevelWarn, StatusIssue, source, message, fields)
}

// Info logs an informational event.
func Info(source, message string, fields map[string]interface{}) {
	defaultSvc.Log(LevelInfo, StatusInfo, source, message, fields)
}

// Errf is a helper that logs a failed operation with an error value.
func Errf(source, message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	defaultSvc.Log(LevelError, StatusFailed, source, message, fields)
}
