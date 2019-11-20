package main

import (
	"context"
	"time"

	"github.com/apex/log"
	"github.com/tierklinik-dobersberg/dxray/pkg/scan"
	"github.com/tierklinik-dobersberg/dxray/pkg/search"
	"github.com/tierklinik-dobersberg/micro/pkg/config"
	"github.com/tierklinik-dobersberg/micro/pkg/service"
)

// StudyIndexer priodically scans a DX-R file-system database
// and provides search functionallity for studies and patients
type StudyIndexer struct {
	*search.Index

	dxr            *DXR
	scanner        *scan.Scanner
	indexPath      string
	repeatFullScan time.Duration
	ticker         *time.Ticker
}

// NewStudyIndexer creates a new study indexer
func NewStudyIndexer(dxr *DXR) *StudyIndexer {
	return &StudyIndexer{
		dxr:       dxr,
		indexPath: "index.bleve",
	}
}

// Init initializes the study indexer
func (s *StudyIndexer) Init() error {
	db, err := s.dxr.Open()
	if err != nil {
		return err
	}

	s.scanner = scan.New(db)
	s.Index, err = search.New(s.indexPath)
	if err != nil {
		return err
	}

	if s.repeatFullScan != 0 {
		log.Infof("scanning database every %s", s.repeatFullScan)

		s.ticker = time.NewTicker(s.repeatFullScan)
		go func() {
			for range s.ticker.C {
				s.FullScan(context.Background())
			}
		}()
	}

	return nil
}

// Directive returns a service configuration directive for
// the indexer
func (s *StudyIndexer) Directive() service.Directive {
	return service.Directive{
		Name: "index",
		Init: func(i *service.Instance, c config.Dispenser) error {
			c.Next()

			if c.NextArg() {
				switch c.Val() {
				case "{":
					c.Unread()
				default:
					s.indexPath = c.Val()
				}
			}

			for c.NextBlock() {
				switch c.Val() {
				case "path":
					if !c.NextArg() {
						return c.ArgErr()
					}
					s.indexPath = c.Val()

				case "every", "repeat":
					if !c.NextArg() {
						return c.ArgErr()
					}
					p, err := time.ParseDuration(c.Val())
					if err != nil {
						return c.SyntaxErr(err.Error())
					}
					s.repeatFullScan = p
				}
			}

			return nil
		},
	}
}

// FullScan scans all studies and updates the index
func (s *StudyIndexer) FullScan(ctx context.Context) error {
	start := time.Now()

	count := 0
	newStudies := 0
	existingStudies := 0

	log.WithField("module", "indexer").Debug("starting full database index scan")

	studies, err := s.scanner.Scan(ctx)
	if err != nil {
		return err
	}

	for study := range studies {
		count++
		new, err := s.Index.Add(study)
		if err != nil {
			log.WithFields(log.Fields{
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
			log.WithFields(log.Fields{
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

	log.WithFields(log.Fields{
		"module": "indexer",
		"total":  count,
		"new":    newStudies,
		"known":  existingStudies,
	}).Infof("Scan finished in %s", duration)

	return nil
}
