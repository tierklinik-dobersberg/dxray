package search

import (
	"errors"
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
)

type (
	// Index is a search index and provides full-text search capabilities.
	// It is backed a blevesearch/bleve index.
	Index struct {
		index bleve.Index
	}

	// StudyDocument holds all keys that should be searchable
	StudyDocument struct {
		Owner       string `json:"owner"`
		Patient     string `json:"patient"`
		Race        string `json:"race"`
		PatientID   string `json:"id"`
		StudyUID    string `json:"uid"`
		Date        string `json:"date"`
		Description string `json:"description"`
	}
)

// New opens an existing search index or creates a new one
func New(path string) (*Index, error) {

	if path == ":memory:" {
		index, err := bleve.NewMemOnly(bleve.NewIndexMapping())
		if err != nil {
			return nil, err
		}
		return &Index{index}, nil
	}

	index, err := bleve.Open(path)
	if err != nil {
		if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
			index, err = bleve.New(path, bleve.NewIndexMapping())
		} else {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	return &Index{index}, nil
}

// Open opens an existing search index
func Open(path string) (*Index, error) {
	index, err := bleve.Open(path)
	if err != nil {
		return nil, err
	}

	return &Index{index}, nil
}

// Count returns the number of documents stored in the index
func (si *Index) Count() (uint64, error) {
	return si.index.DocCount()
}

// Add adds a new study to the search index
func (si *Index) Add(s fsdb.Study) (bool, error) {
	key := getKey(s)

	d, err := si.index.Document(key)
	if err != nil {
		return false, err
	}

	if d == nil {
		model, err := LoadStudy(s)
		if err != nil {
			return false, err
		}
		return true, si.index.Index(key, model)
	}

	return false, nil
}

// Search search all indexed studies for term
func (si *Index) Search(term string) ([]string, error) {
	query := bleve.NewQueryStringQuery(term)
	search := bleve.NewSearchRequest(query)

	results, err := si.index.Search(search)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(results.Hits))
	for _, h := range results.Hits {
		ids = append(ids, h.ID)
	}

	return ids, nil
}

// getKey returns the byte representation of a study key
func getKey(s fsdb.Study) string {
	return fmt.Sprintf("%s/%s", s.Volume().Name(), s.Name())
}

// LoadStudy loads the study s and returns the study document representation
// This method is mainly exporeted for debugging reasons and my vanish at any
// time
func LoadStudy(s fsdb.Study) (*StudyDocument, error) {
	if err := s.Load(); err != nil {
		return nil, err
	}

	model, _ := s.Model()

	var desc []string

	if model.Patient.Visit.Study.Description != "" {
		desc = append(desc, model.Patient.Visit.Study.Description)
	}

	for _, s := range model.Patient.Visit.Study.Series {
		if s.Description != "" {
			desc = append(desc, s.Description)
		}
	}

	return &StudyDocument{
		Owner:       model.Patient.OwnerName(),
		Patient:     model.Patient.AnimalName(),
		Race:        model.Patient.AnimalRace(),
		PatientID:   model.Patient.ID,
		StudyUID:    model.Patient.Visit.Study.UID,
		Date:        model.Patient.Visit.Study.Date,
		Description: strings.Join(desc, "\n"),
	}, nil
}

// Get opens the study identified by key from the database
func Get(key string, db fsdb.DB) (fsdb.Study, error) {
	parts := strings.Split(key, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid study key")
	}

	vol, err := db.OpenVolumeByName(parts[0])
	if err != nil {
		return nil, err
	}

	stdy, err := vol.OpenStudyByName(parts[1])
	if err != nil {
		return nil, err
	}

	return stdy, nil
}
