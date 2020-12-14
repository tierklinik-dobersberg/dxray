package app

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
	"github.com/tierklinik-dobersberg/dxray/internal/index"
	"github.com/tierklinik-dobersberg/service/server"
)

type contextKey string

const appContextKey = contextKey("dxray:app")

// App holds dependencies that are required throught the
// dxray.
type App struct {
	FsDB    fsdb.DB
	Indexer *index.StudyIndexer
}

// New returns a new App.
func New(db fsdb.DB, indexer *index.StudyIndexer) *App {
	return &App{
		FsDB:    db,
		Indexer: indexer,
	}
}

// With adds app to the context.
func With(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appContextKey, app)
}

// From returns the app added to ctx previously by using
// With.
func From(c *gin.Context) *App {
	app, _ := c.Request.Context().Value(appContextKey).(*App)
	return app
}

// AddToRequest returns a (service/server).PreHandlerFunc that
// adds app to each incoming request. Use From() to retrieve
// *app from a gin.Context afterwards.
func AddToRequest(app *App) server.PreHandlerFunc {
	return func(req *http.Request) *http.Request {
		newCtx := With(req.Context(), app)
		return req.Clone(newCtx)
	}
}
