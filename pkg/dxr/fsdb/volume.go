package fsdb

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type (
	// Volume abstracts opertions on a ORconsoleDB volume
	Volume interface {
		// Name returns the name of the volume
		Name() string

		// Path returns the path to the volume
		Path() string

		// Index returns the index of the volume or -1 if
		// the name of the volume does not follow the standard
		// volume naming schema (VOLxxxxx where x is a number)
		Index() int

		// CountStudies returns the number of studies stored in
		// the volume. It seems like this is always capped at
		// 1000 studies per volume
		CountStudies() (int, error)

		// Studies returns a slice of all study names stored in
		// the volume
		Studies() ([]string, error)

		// First returns the first study in the volume
		First() (Study, error)

		// Last returns the last study in the volume
		Last() (Study, error)

		// OpenStudyByName opens the study with the given name
		OpenStudyByName(name string) (Study, error)

		// ForEachStudy executes fn for each study inside the volume
		ForEachStudy(fn func(s Study) error) error
	}

	// volume implements the Volume interface
	volume struct {
		db   *db
		name string
	}
)

// Name returns the name of the volume and implements
// the Volume interface
func (v *volume) Name() string {
	return v.name
}

// Path returns the path to the volume and implements
// the Volume interface
func (v *volume) Path() string {
	return filepath.Join(v.db.rootPath, v.name)
}

// Index returns the index of the volume
// or -1 if it doesn't follow the normal naming schema.
// It implements the Volume interface
func (v *volume) Index() int {
	// volumes follow the naming schema VOLxxxxx where x is a number
	if len(v.Name()) < 8 {
		return -1
	}

	paddedNum := strings.Trim(v.Name()[3:], "0")
	if len(paddedNum) == 0 {
		return 0
	}

	i, err := strconv.Atoi(paddedNum)
	if err != nil {
		return -1
	}

	return i
}

// CountSutdies returns the number of studies inside the volume
// It implements the Volume interface
func (v *volume) CountStudies() (int, error) {
	count := 0
	files, err := ioutil.ReadDir(v.Path())
	if err != nil {
		return 0, err
	}

	// we should only count diretories here
	for _, f := range files {
		if f.IsDir() {
			count++
		}
	}

	return count, nil
}

// OpenStudyByName opens the study with the given name
// It implements the Volume interface
func (v *volume) OpenStudyByName(name string) (Study, error) {
	studyPath := filepath.Join(v.Path(), name)
	stat, err := os.Stat(studyPath)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("invalid study. expected directory but got a file")
	}

	return &study{
		db:   v.db,
		vol:  v,
		name: name,
	}, nil
}

// Studies returns all studies names stored inside the volume
// It implements the Volume interface
func (v *volume) Studies() ([]string, error) {
	files, err := ioutil.ReadDir(v.Path())
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		// Studies must be directories
		if !f.IsDir() {
			continue
		}

		// we could check for the correct study name schema here
		names = append(names, f.Name())
	}

	return names, nil
}

// ForEachStudy executes fn for each study inside the volume
// It implements the Volume interface
func (v *volume) ForEachStudy(fn func(Study) error) error {
	studies, err := v.Studies()
	if err != nil {
		return err
	}

	for _, sname := range studies {
		s, err := v.OpenStudyByName(sname)
		if err != nil {
			return err
		}

		if err := fn(s); err != nil {
			return err
		}
	}
	return nil
}

// First returns the first study in the volume
// It implements the Volume interface
// TODO(ppacher): we currently rely on the fact that the
// study name are sorted correctly
func (v *volume) First() (Study, error) {
	all, err := v.Studies()
	if err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, nil
	}

	return v.OpenStudyByName(all[0])
}

// Last returns the last study in the volume
// It implements the Volume interface
// TODO(ppacher): we currently rely on the fact that the
// study name are sorted correctly
func (v *volume) Last() (Study, error) {
	all, err := v.Studies()
	if err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, nil
	}

	return v.OpenStudyByName(all[len(all)-1])
}
