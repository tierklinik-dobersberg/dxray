// Package fsdb abstracts access to the folder structure used in DX-R ORconsoleDB
package fsdb

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
)

type (
	// DB abstracts access to studies and series stored ina ORconsoleDB folder
	DB interface {
		// VolumeNames returns a list of volume names
		VolumeNames() ([]string, error)

		// OpenVolumeByName opens the database volume with the given name.
		// Note that ORconsoleDB volumes normally start with VOL
		OpenVolumeByName(name string) (Volume, error)

		// OpenVolumeByIdx opens the database volume with the given index
		OpenVolumeByIdx(id int) (Volume, error)

		// ForEachVolume calls the given function for each volume inside
		// the database. Any non-nil return error will stop the iteration
		ForEachVolume(func(vol Volume) error) error
	}

	// db implements the DB interface
	db struct {
		rootPath string // path to the ORconsoleDB folder
		l        log.Interface
	}
)

// New creates a new DB from the given ORconsoleDB path
func New(path string, logger log.Interface) (DB, error) {
	if logger == nil {
		logger = log.Log
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, errors.New("expected a directory but got a file")
	}

	return &db{
		rootPath: path,
		l:        logger,
	}, nil
}

// VolumeNames returns a list of volume names in the ORconsoleDB
// directory
func (d *db) VolumeNames() ([]string, error) {
	files, err := ioutil.ReadDir(d.rootPath)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		fname := f.Name()

		if !strings.HasPrefix(fname, "VOL") {
			d.l.WithField("name", fname).Warnf("volumes must have a VOL prefix")
			continue
		}

		names = append(names, fname)
	}
	return names, nil
}

// OpenVolumeByName opens the database volume with the given name.
// It implements the DB interface
func (d *db) OpenVolumeByName(name string) (Volume, error) {
	volPath := filepath.Join(d.rootPath, name)
	stat, err := os.Stat(volPath)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("invalid volume. Expected a directory but got a file")
	}

	return &volume{
		name: name,
		db:   d,
	}, nil
}

// OpenVolumeByIdx opens the database volume with the given index.
// It implements the DB interface
func (d *db) OpenVolumeByIdx(idx int) (Volume, error) {
	idxStr := fmt.Sprintf("VOL%05d", idx)
	return d.OpenVolumeByName(idxStr)
}

// ForEachVolume calls fn for each volume in the database.
// It implements the DB interface
func (d *db) ForEachVolume(fn func(Volume) error) error {
	volumes, err := d.VolumeNames()
	if err != nil {
		return err
	}

	for _, name := range volumes {
		vol, err := d.OpenVolumeByName(name)
		if err != nil {
			return err
		}

		if err := fn(vol); err != nil {
			return err
		}
	}
	return nil
}
