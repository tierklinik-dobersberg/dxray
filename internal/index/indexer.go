package index

import (
	"context"
	"time"

	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/internal/scan"
	"github.com/tierklinik-dobersberg/dxray/internal/search"
	"github.com/tierklinik-dobersberg/logger"
)

// StudyIndexer priodically scans a DX-R file-system database
// and provides search functionallity for studies and patients
type StudyIndexer struct {
	*search.Index

	db             fsdb.DB
	scanner        *scan.Scanner
	indexPath      string
	repeatFullScan time.Duration
	ticker         *time.Ticker
}

// NewStudyIndexer creates a new study indexer
func NewStudyIndexer(db fsdb.DB, path string, repeat time.Duration) (*StudyIndexer, error) {
	if path == "" {
		path = "index.bleve"
	}

	idx := &StudyIndexer{
		db:             db,
		indexPath:      path,
		repeatFullScan: repeat,
	}

	if err := idx.init(); err != nil {
		return nil, err
	}

	return idx, nil
}

// Init initializes the study indexer
func (s *StudyIndexer) init() error {
	var err error

	s.scanner = scan.New(s.db)
	s.Index, err = search.New(s.indexPath)
	if err != nil {
		return err
	}

	if s.repeatFullScan != 0 {
		logger.DefaultLogger().Infof("scanning database every %s", s.repeatFullScan)

		s.ticker = time.NewTicker(s.repeatFullScan)
		go func() {
			for range s.ticker.C {
				s.FullScan(context.Background())
			}
		}()
	}

	return nil
}

// FullScan scans all studies and updates the index
func (s *StudyIndexer) FullScan(ctx context.Context) error {
	log := logger.From(ctx)

	start := time.Now()

	count := 0
	newStudies := 0
	existingStudies := 0

	log.WithFields(logger.Fields{
		"module": "indexer",
	}).Info("starting full database index scan")

	studies, err := s.scanner.Scan(ctx)
	if err != nil {
		return err
	}

	for study := range studies {
		count++
		new, err := s.Index.Add(study)
		if err != nil {
			log.WithFields(logger.Fields{
				"error":  err.Error(),
				"module": "indexer",
				"study":  study.Name(),
				"volume": study.Volume().Name(),
			}).Errorf("failed to index study")
		} else {
			if new {
				newStudies++
			} else {
				existingStudies++
			}
		}

		if count%100 == 0 && time.Now().Sub(start) > 5*time.Second {
			log.WithFields(logger.Fields{
				"module":   "indexer",
				"study":    study.Name(),
				"volume":   study.Volume().Name(),
				"duration": time.Now().Sub(start).Round(time.Second),
			}).Infof("scanned %d studies so far ...", count)
		}
	}

	round := 500 * time.Millisecond
	duration := time.Now().Sub(start)
	if duration < time.Second {
		round = time.Millisecond
	}
	duration = duration.Round(round)

	log.WithFields(logger.Fields{
		"module": "indexer",
		"total":  count,
		"new":    newStudies,
		"known":  existingStudies,
	}).Infof("Scan finished in %s", duration)

	return nil
}
