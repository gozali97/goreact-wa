package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Store manages media files on the local filesystem. The database only stores
// the relative path (e.g. /storage/images/xxx.jpg), so the backing store can
// later be swapped for S3/MinIO/R2 without schema changes.
type Store struct {
	root string
}

// New creates a Store rooted at the given path and ensures subdirs exist.
func New(root string) (*Store, error) {
	for _, sub := range []string{"images", "docs", "media"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir %s: %w", sub, err)
		}
	}
	return &Store{root: root}, nil
}

// Root returns the storage root directory.
func (s *Store) Root() string { return s.root }

// SaveImage writes image bytes and returns the public relative URL path.
func (s *Store) SaveImage(data []byte, ext string) (string, error) {
	return s.save("images", data, ext)
}

// SaveDocument writes document bytes and returns the public relative URL path.
func (s *Store) SaveDocument(data []byte, name string) (string, error) {
	ext := filepath.Ext(name)
	return s.save("docs", data, ext)
}

// SaveMedia writes arbitrary media (video/audio/sticker) bytes with the given
// extension and returns the public relative URL path.
func (s *Store) SaveMedia(data []byte, ext string) (string, error) {
	return s.save("media", data, ext)
}

func (s *Store) save(sub string, data []byte, ext string) (string, error) {
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "bin"
	}
	name := fmt.Sprintf("%d_%s.%s", time.Now().Unix(), uuid.NewString()[:8], ext)
	full := filepath.Join(s.root, sub, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	// Public path served by the HTTP layer.
	return fmt.Sprintf("/storage/%s/%s", sub, name), nil
}

// ─── Monitoring & management ──────────────────────────────

// FileInfo describes a stored media file for the file manager.
type FileInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"` // images | docs | media
	Path     string `json:"path"`     // public /storage/... URL path
	Size     int64  `json:"size"`
	ModTime  int64  `json:"mod_time"` // unix seconds
}

// CategoryStat summarizes one storage subdirectory.
type CategoryStat struct {
	Category string `json:"category"`
	Files    int    `json:"files"`
	Size     int64  `json:"size"`
}

// categories are the managed subdirectories.
var categories = []string{"images", "docs", "media"}

// IsValidCategory reports whether c is a managed category.
func IsValidCategory(c string) bool {
	for _, x := range categories {
		if x == c {
			return true
		}
	}
	return false
}

// Categories returns the managed subdirectory names.
func Categories() []string { return categories }

// Stats returns per-category file counts and total bytes.
func (s *Store) Stats() ([]CategoryStat, int64, error) {
	var out []CategoryStat
	var total int64
	for _, cat := range categories {
		dir := filepath.Join(s.root, cat)
		entries, err := os.ReadDir(dir)
		if err != nil {
			out = append(out, CategoryStat{Category: cat})
			continue
		}
		var count int
		var size int64
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			count++
			size += info.Size()
		}
		out = append(out, CategoryStat{Category: cat, Files: count, Size: size})
		total += size
	}
	return out, total, nil
}

// ListFiles returns files in a category sorted newest-first.
func (s *Store) ListFiles(category string) ([]FileInfo, error) {
	if !IsValidCategory(category) {
		return nil, fmt.Errorf("invalid category")
	}
	dir := filepath.Join(s.root, category)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, err
	}
	files := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:     e.Name(),
			Category: category,
			Path:     fmt.Sprintf("/storage/%s/%s", category, e.Name()),
			Size:     info.Size(),
			ModTime:  info.ModTime().Unix(),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ModTime > files[j].ModTime })
	return files, nil
}

// safeName rejects path traversal and ensures a plain file name.
func safeName(name string) (string, error) {
	clean := filepath.Base(name)
	if clean != name || clean == "." || clean == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid file name")
	}
	return clean, nil
}

// DeleteFile removes a single file from a category.
func (s *Store) DeleteFile(category, name string) error {
	if !IsValidCategory(category) {
		return fmt.Errorf("invalid category")
	}
	clean, err := safeName(name)
	if err != nil {
		return err
	}
	full := filepath.Join(s.root, category, clean)
	if err := os.Remove(full); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

// ClearCategory deletes every file in a category and returns the count removed.
func (s *Store) ClearCategory(category string) (int, error) {
	if !IsValidCategory(category) {
		return 0, fmt.Errorf("invalid category")
	}
	dir := filepath.Join(s.root, category)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
			removed++
		}
	}
	return removed, nil
}

// ClearAll empties every managed category and returns the total files removed.
func (s *Store) ClearAll() (int, error) {
	total := 0
	for _, cat := range categories {
		n, err := s.ClearCategory(cat)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// DeleteOlderThan removes files in all managed categories whose modtime is
// before the cutoff, returning the number removed and bytes freed.
func (s *Store) DeleteOlderThan(cutoff time.Time) (int, int64, error) {
	var removed int
	var freed int64
	for _, cat := range categories {
		dir := filepath.Join(s.root, cat)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
					removed++
					freed += info.Size()
				}
			}
		}
	}
	return removed, freed, nil
}
