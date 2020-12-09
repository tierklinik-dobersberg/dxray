package main

import (
	"context"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/tierklinik-dobersberg/dxray/internal/index"
	"github.com/tierklinik-dobersberg/micro/pkg/api"
	"github.com/tierklinik-dobersberg/micro/pkg/auth/authn"
	"github.com/tierklinik-dobersberg/micro/pkg/auth/authz"
	"github.com/tierklinik-dobersberg/micro/pkg/config"
	"github.com/tierklinik-dobersberg/micro/pkg/metrics"
	"github.com/tierklinik-dobersberg/micro/pkg/server"
	"github.com/tierklinik-dobersberg/micro/pkg/service"
)

func main() {
	log.SetHandler(cli.New(os.Stdout))

	dxr := &DXR{}

	instance := service.NewInstance(service.Config{
		Name:        "dxray",
		InputLoader: config.FileLoader("Configfile"),
		Directives: []service.Directive{
			metrics.Directive,
			server.Directive(),
			authn.Directive(),
			authz.Directive(),
			dxr.Directive(),
		},
		Modules: []api.Module{API},
	})

	db, err := dxr.Open()
	if err != nil {
		log.WithError(err).Fatal("failed to open dxr path")
	}
	indexer, err := index.NewStudyIndexer(db, "", time.Minute*2)
	if err != nil {
		log.WithError(err).Fatal("failed to create study indexer")
	}

	// make sure we have our DXR and indexer provided via the router
	instance.AddProvider(ContextKeyDXR, dxr)
	instance.AddProvider(ContextKeyIndexer, indexer)

	if err := instance.InitRouter(); err != nil {
		log.WithError(err).Fatal("failed to prepare router")
	}

	count, err := indexer.Count()
	if err != nil {
		log.WithError(err).Fatal("failed to get index count")
	}
	log.Infof("study index contains %d studies", count)

	if err := indexer.FullScan(context.Background()); err != nil {
		log.WithError(err).Fatal("failed to perform initial full scan")
	}

	if err := server.Serve(instance); err != nil {
		log.WithError(err).Fatal("failed to listen for connection")
	}
}
