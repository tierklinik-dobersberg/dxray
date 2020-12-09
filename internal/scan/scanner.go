package scan

import (
	"context"
	"errors"

	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
)

var (
	// ErrScanRunning is returned it a scanner is already running.
	ErrScanRunning = errors.New("scan is already running")
)

// Scanner scans a DX-R fsdb for all studies stored
// inside.
type Scanner struct {
	db fsdb.DB
}

// New returns a new scanner
func New(db fsdb.DB) *Scanner {
	return &Scanner{
		db: db,
	}
}

// Scan scans the whole fsdb and returns each study found.
// If ctx is cancelled the scan will be stopped.
func (s *Scanner) Scan(ctx context.Context) (<-chan fsdb.Study, error) {
	ch := make(chan fsdb.Study)

	go s.scan(ctx, ch)

	return ch, nil
}

// ScanVolume scans a given volume.
func (s *Scanner) ScanVolume(ctx context.Context, vol fsdb.Volume) (chan fsdb.Study, error) {
	ch := make(chan fsdb.Study)

	go func() {
		defer close(ch)
		s.scanVolume(ctx, ch, vol)
	}()

	return ch, nil
}

// ScanVolumeName scans the volume with a given name.
func (s *Scanner) ScanVolumeName(ctx context.Context, name string) (chan fsdb.Study, error) {
	vol, err := s.db.OpenVolumeByName(name)
	if err != nil {
		return nil, err
	}

	return s.ScanVolume(ctx, vol)
}

// ScanVolumeID scans the volume with ID.
func (s *Scanner) ScanVolumeID(ctx context.Context, idx int) (chan fsdb.Study, error) {
	vol, err := s.db.OpenVolumeByIdx(idx)
	if err != nil {
		return nil, err
	}

	return s.ScanVolume(ctx, vol)
}

func (s *Scanner) scan(ctx context.Context, ch chan fsdb.Study) {
	defer close(ch)

	s.db.ForEachVolume(func(vol fsdb.Volume) error {
		return s.scanVolume(ctx, ch, vol)
	})
}

func (s *Scanner) scanVolume(ctx context.Context, ch chan fsdb.Study, vol fsdb.Volume) error {
	return vol.ForEachStudy(func(stdy fsdb.Study) error {
		select {
		case ch <- stdy:

		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})
}
