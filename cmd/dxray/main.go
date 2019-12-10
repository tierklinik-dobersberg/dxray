package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr/v2"
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
	indexer := NewStudyIndexer(dxr)

	instance := service.NewInstance(service.Config{
		Name:        "dxray",
		InputLoader: config.FileLoader("Configfile"),
		Directives: []service.Directive{
			metrics.Directive,
			server.Directive(),
			authn.Directive(),
			authz.Directive(),
			dxr.Directive(),
			indexer.Directive(),
		},
		Modules: []api.Module{API},
	})

	// make sure we have our DXR and indexer provided via the router
	instance.AddProvider(ContextKeyDXR, dxr)
	instance.AddProvider(ContextKeyIndexer, indexer)

	if err := instance.InitRouter(); err != nil {
		log.WithError(err).Fatal("failed to prepare router")
	}

	// we need to serve static files for the ohiv viewer
	uiBox := &fsBox{packr.New("UI", "../Viewers/platform/viewer/dist/")}
	instance.Router.Use(static.Serve("", uiBox), func(ctx *gin.Context) {
		// we skip everything with a /v1 prefix
		if strings.HasPrefix(ctx.Request.URL.Path, "/v1") {
			return
		}

		index, err := uiBox.Open(static.INDEX)
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		defer index.Close()

		fs, err := index.Stat()
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}

		ctx.DataFromReader(http.StatusOK, fs.Size(), "text/html", index, map[string]string{})
	})

	if err := indexer.Init(); err != nil {
		log.WithError(err).Fatal("failed to initialize database index")
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
