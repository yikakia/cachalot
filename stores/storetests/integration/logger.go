package integration

import (
	"github.com/testcontainers/testcontainers-go/log"
)

type noopLogger struct{}

func (n noopLogger) Printf(format string, v ...any) {
	return
}

var _ log.Logger = (*noopLogger)(nil)
