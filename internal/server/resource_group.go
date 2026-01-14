package server

import (
	"go.uber.org/zap"
	"io"
)

type ResourceGroup struct {
	closers []io.Closer
	log     *zap.Logger
}

func NewResourceGroup(log *zap.Logger) *ResourceGroup {
	return &ResourceGroup{
		closers: []io.Closer{},
		log:     log,
	}
}

func (rg *ResourceGroup) Register(c io.Closer) {
	rg.closers = append(rg.closers, c)
}

func (rg *ResourceGroup) CloseAll() error {
	for i, closer := range rg.closers {
		if err := closer.Close(); err != nil {
			rg.log.Error("Resource close failed",
				zap.Int("resource_index", i),
				zap.Error(err),
			)
			return err
		}
	}

	return nil
}
