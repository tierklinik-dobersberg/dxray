package main

import (
	"context"
	"fmt"
	"os"
	"time"

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

	ctx := context.Background()

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
		logger.Fatalf(ctx, "failed to bootstap: %s", err)
	}

	// Ensure the configured database path actually exists and is
	// a directory.
	if err := ensureDirectory(cfg.DatabasePath); err != nil {
		logger.Fatalf(ctx, "failed to bootstap: %s", err)
	}

	// Create a new fsdb for the given database path
	db, err := fsdb.New(cfg.DatabasePath, logger.DefaultLogger().WithFields(logger.Fields{
		"fsdb": cfg.DatabasePath,
	}))
	if err != nil {
		logger.Fatalf(ctx, "failed to bootstap: %s", err)
	}

	// Create a new study-indxer that scans for new studies every
	// two minutes.
	indexer, err := index.NewStudyIndexer(db, "", time.Minute*2)
	if err != nil {
		logger.Fatalf(ctx, "failed to create study indexer: %s", err)
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
		logger.Fatalf(ctx, "failed to get index count: %s", err)
	}
	logger.Infof(ctx, "study index contains %d studies", count)

	// Perform a new full scan so we start with an up-to-data
	// study index.
	if err := indexer.FullScan(context.Background()); err != nil {
		logger.Fatalf(ctx, "failed to perform initial full scan: %s", err)
	}

	// Finally, start serving our API.
	if err := instance.Serve(); err != nil {
		logger.Fatalf(ctx, "failed to serve: %s", err)

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
