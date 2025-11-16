package server_test

import (
	"testing"
	"time"

	"github.com/b-sea/go-server/mock"
	"github.com/b-sea/go-server/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSetVersion(t *testing.T) {
	testServer := server.New(zerolog.Nop(), mock.NewNoOp())
	server.SetVersion("special-test-version")(testServer)
	assert.Equal(t, "special-test-version", testServer.Version())
}

func TestSetPort(t *testing.T) {
	testServer := server.New(zerolog.Nop(), mock.NewNoOp())
	server.SetPort(4567)(testServer)

	assert.Equal(t, ":4567", testServer.Addr())
}

func TestSetReadTimeout(t *testing.T) {
	testServer := server.New(zerolog.Nop(), mock.NewNoOp())
	server.SetReadTimeout(time.Hour)(testServer)
	assert.Equal(t, time.Hour, testServer.ReadTimeout())

	testServer = server.New(zerolog.Nop(), mock.NewNoOp())
	server.SetReadTimeout(0)(testServer)
	assert.Equal(t, 10*time.Second, testServer.ReadTimeout())
}

func TestSetWriteTimeout(t *testing.T) {
	testServer := server.New(zerolog.Nop(), mock.NewNoOp())
	server.SetWriteTimeout(time.Hour)(testServer)
	assert.Equal(t, time.Hour, testServer.WriteTimeout())

	testServer = server.New(zerolog.Nop(), mock.NewNoOp())
	server.SetWriteTimeout(0)(testServer)
	assert.Equal(t, 10*time.Second, testServer.WriteTimeout())
}
