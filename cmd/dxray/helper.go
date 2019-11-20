package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/tierklinik-dobersberg/dxray/pkg/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/pkg/search"
)

func runREPL(index *search.Index, db fsdb.DB) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			log.WithError(err).Fatal("bye bye")
		}

		if strings.HasPrefix(text, "get") {
			path := strings.TrimSpace(strings.TrimPrefix(text, "get"))
			s, err := openStudy(db, path)
			if err != nil {
				continue
			}

			doc, _ := s.Model()

			blob, err := json.MarshalIndent(doc, "", "  ")
			if err != nil {
				log.WithError(err).Errorf("failed to open study")
				continue
			}

			fmt.Println(string(blob))
		} else if strings.HasPrefix(text, "show") {
			path := strings.TrimSpace(strings.TrimPrefix(text, "show"))
			s, err := openStudy(db, path)
			if err != nil {
				continue
			}

			doc, _ := search.LoadStudy(s)
			blob, err := json.MarshalIndent(doc, "", "  ")
			if err != nil {
				log.WithError(err).Errorf("failed to open study")
				continue
			}

			fmt.Println(string(blob))

		} else {
			results, err := index.Search(text)
			if err != nil {
				log.WithError(err).Fatal("bye bye")
			}

			for i, r := range results {
				fmt.Printf("\t%02d. %s\n", i, r)
			}
		}
	}
}

func openStudy(db fsdb.DB, path string) (fsdb.Study, error) {
	path = strings.TrimSpace(path)
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid input")
	}

	vol, err := db.OpenVolumeByName(parts[0])
	if err != nil {
		log.WithError(err).Errorf("failed to open volume")
		return nil, err
	}

	s, err := vol.OpenStudyByName(parts[1])
	if err != nil {
		log.WithError(err).Errorf("failed to open study")
		return nil, err
	}

	err = s.Load()
	if err != nil {
		log.WithError(err).Errorf("failed to open study")
		return nil, err
	}

	return s, nil
}
