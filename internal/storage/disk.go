package storage

import (
	"path/filepath"

	"github.com/shirou/gopsutil/v4/disk"
)

// DiskStat describes the disk/volume the storage path lives on.
type DiskStat struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// Disk returns usage stats for the filesystem that holds the storage root.
func (s *Store) Disk() (*DiskStat, error) {
	abs, err := filepath.Abs(s.root)
	if err != nil {
		abs = s.root
	}
	u, err := disk.Usage(abs)
	if err != nil {
		return nil, err
	}
	return &DiskStat{
		Path:        u.Path,
		Total:       u.Total,
		Used:        u.Used,
		Free:        u.Free,
		UsedPercent: u.UsedPercent,
	}, nil
}
