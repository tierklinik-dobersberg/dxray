package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/dxray/internal/app"
	"github.com/tierklinik-dobersberg/dxray/internal/ohif"
	"github.com/tierklinik-dobersberg/dxray/internal/search"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/server"
)

// SearchStudiesEndpoint allows searching studies by querying
// the study-indexer (bleeve index).
//
// GET /api/v1/search
func SearchStudiesEndpoint(grp gin.IRouter) {
	grp.GET("search", func(c *gin.Context) {
		log := logger.From(c.Request.Context())

		appCtx := app.From(c)
		if appCtx == nil {
			return
		}

		term := c.Query("q")
		results, err := appCtx.Indexer.Search(term)
		if err != nil {
			server.AbortRequest(c, http.StatusInternalServerError, err)
			return
		}

		models := make([]*ohif.StudyJSON, 0, len(results))
		for _, key := range results {
			s, err := search.Get(key, appCtx.FsDB)
			if err != nil {
				log.WithFields(logger.Fields{
					"error": err.Error(),
					"key":   key,
				}).Errorf("failed to open study")
				continue
			}

			m, err := ohif.JSONFromDXR(c.Request.Context(), s, createStudyURLFactory(c), false)
			if err != nil {
				log.WithFields(logger.Fields{
					"error": err.Error(),
					"key":   key,
				}).Errorf("failed to get JSON representaion")
				continue
			}

			models = append(models, m)
		}

		c.JSON(http.StatusOK, models)
	})
}
