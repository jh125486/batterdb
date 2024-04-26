package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/handlers"
)

func TestService_Start(t *testing.T) {
	t.Parallel()
	svc, _ := handlers.New()
	go func() {
		_ = svc.Start(0)
	}()
	require.NoError(t, svc.Shutdown())
}
