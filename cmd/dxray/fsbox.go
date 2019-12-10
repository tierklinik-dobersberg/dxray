package main

import (
	"path"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gobuffalo/packr/v2"
)

type fsBox struct {
	*packr.Box
}

func (b *fsBox) Exists(prefix, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix+"/"); len(p) < len(filepath) {
		has := b.Has(p)
		if !has {
			if b.Has(path.Join(p, static.INDEX)) {
				return true
			}
		}
		return has
	}
	return false
}
