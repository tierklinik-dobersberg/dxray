package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tierklinik-dobersberg/dxray/internal/dxr/fsdb"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/micro/pkg/config"
	"github.com/tierklinik-dobersberg/micro/pkg/service"
)

// DXR holds configuration and utility methods for interacting
// the a DX-R file-system
type DXR struct {
	path string
}

// Directive returns a service.Directive to configure the DXR
func (dxr *DXR) Directive() service.Directive {
	return service.Directive{
		Name: "dxr",
		Init: func(i *service.Instance, c config.Dispenser) error {
			c.Next()

			if c.NextArg() {
				switch c.Val() {
				case "{":
					c.Unread()
				default:
					dxr.path = filepath.Clean(c.Val())
				}
			}

			if err := ensureDirectory(dxr.path); err != nil {
				return err
			}

			if c.Next() {
				return c.SyntaxErr("Unexpected token or duplicated dxr config block")
			}

			return nil
		},
	}
}

// Open opens the DX-R file-system
func (dxr *DXR) Open() (fsdb.DB, error) {
	return fsdb.New(dxr.path, logger.DefaultLogger().WithFields(logger.Fields{
		"fsdb": dxr.path,
	}))
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
