package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/gin-gonic/gin"
	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/dxray/internal/api"
	"github.com/tierklinik-dobersberg/dxray/internal/app"
	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/internal/index"
	"github.com/tierklinik-dobersberg/dxray/internal/schema"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/service"
)

func main() {
	var cfg struct {
		schema.Config `section:"Global"`
	}

	instance, err := service.Boot(service.Config{
		ConfigFileName: "dxray.conf",
		ConfigFileSpec: conf.FileSpec{
			"global": schema.ConfigSpec,
		},
		ConfigTarget: &cfg,
		RouteSetupFunc: func(grp gin.IRouter) error {
			api.ListStudiesEndpoint(grp)
			api.OHIFEndpoint(grp)
			api.SearchStudiesEndpoint(grp)
			return nil
		},
	})
	if err != nil {
		logger.Errorf("failed to bootstap: %s", err)
		os.Exit(1)
	}

	// Ensure the configured database path actually exists and is
	// a directory.
	if err := ensureDirectory(cfg.DatabasePath); err != nil {
		logger.Errorf("failed to bootstap: %s", err)
		os.Exit(1)
	}

	// Create a new fsdb for the given database path
	db, err := fsdb.New(cfg.DatabasePath, logger.DefaultLogger().WithFields(logger.Fields{
		"fsdb": cfg.DatabasePath,
	}))
	if err != nil {
		logger.Errorf("failed to bootstap: %s", err)
		os.Exit(1)
	}

	// Create a new study-indxer that scans for new studies every
	// two minutes.
	indexer, err := index.NewStudyIndexer(db, "", time.Minute*2)
	if err != nil {
		logger.Errorf("failed to create study indexer: %s", err)
		os.Exit(1)
	}

	// Prepare the application context that is passed to each
	// api endpoint.
	appCtx := app.New(db, indexer)
	instance.Server().WithPreHandler(
		app.AddToRequest(appCtx),
	)

	// Get the number of currently sotred studies.
	count, err := indexer.Count()
	if err != nil {
		logger.Errorf("failed to get index count: %s", err)
		os.Exit(1)
	}
	logger.Infof("study index contains %d studies", count)

	// Perform a new full scan so we start with an up-to-data
	// study index.
	if err := indexer.FullScan(context.Background()); err != nil {
		log.Errorf("failed to perform initial full scan: %s", err)
		os.Exit(1)
	}

	// Finally, start serving our API.
	if err := instance.Serve(); err != nil {
		log.Errorf("failed to serve: %s", err)
		os.Exit(1)

	}
}

func ensureDirectory(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("expected a directory")
	}

	return nil
}
