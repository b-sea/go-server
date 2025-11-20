package mock

import (
	"github.com/b-sea/go-server/server"
)

var (
	_ server.HealthChecker = (*HealthCheck)(nil)
)

type HealthCheck struct {
	Err error
}

func (m *HealthCheck) HealthCheck() error {
	return m.Err
}
