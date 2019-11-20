package fsdb

import (
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/tierklinik-dobersberg/dxray/pkg/dxr/models"
)

type (
	// Study abstracts access to study folders stored inside ORconsoleDB
	// volumes
	Study interface {
		// Name returns the name of the study
		Name() string

		// Path returns the filesystem path of the study
		Path() string

		// Index returns the index of the study
		// by parsing the study folder name
		Index() int

		// Volume returns the ORconsoleDB volume that
		// contains the study. Note that this may return
		// nil if the volume of the study is not known
		Volume() Volume

		// Load tries to parse the study.xml file
		Load() error

		// Model returns the ImageList model and a boolean indicating
		// if it has already been loaded
		Model() (models.ImageList, bool)

		// RealPath returns the real path of a file referenced in the study
		RealPath(p string) string
	}

	// study implements the Study interface
	study struct {
		l     sync.Mutex
		name  string
		db    *db
		vol   *volume
		model *models.ImageList
	}
)

// Name returns the name of the study and implements
// the Study interface
func (s *study) Name() string {
	return s.name
}

// Path returns the path of the study
// and implements the Study interface
func (s *study) Path() string {
	return filepath.Join(s.vol.Path(), s.name)
}

// Index returns the index of the study inside the
// volume and implements the Study interfac
func (s *study) Index() int {
	parts := strings.Split(s.name, "_")
	if len(parts) != 2 {
		return -1
	}

	i, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}

	return i
}

// Volume returns the ORconsoleDB volume
// that contains the study. It implements
// the Study interface.
func (s *study) Volume() Volume {
	return s.vol
}

// Load tries to parse the study.xml file
func (s *study) Load() error {
	s.l.Lock()
	defer s.l.Unlock()

	return s.load()
}

// Model returns the ImageList model and a boolean indicating
// if it has already been loaded
func (s *study) Model() (models.ImageList, bool) {
	if s.model == nil {
		return models.ImageList{}, false
	}

	return *s.model, true
}

func (s *study) load() error {
	path := filepath.Join(s.Path(), "study.xml")
	model, err := models.FromFile(path)
	if err != nil {
		return err
	}

	s.model = model

	return nil
}

func (s *study) ensureLoaded() error {
	if s.model == nil {
		return s.load()
	}

	return nil
}

func (s *study) RealPath(p string) string {
	slashed := filepath.ToSlash(p)
	lower := strings.ToLower(slashed)
	prefix := "/dicompacs/orconsoledb/"
	if !strings.HasPrefix(lower, prefix) {
		return slashed
	}

	return filepath.Join(s.db.rootPath, slashed[len(prefix):])
}
