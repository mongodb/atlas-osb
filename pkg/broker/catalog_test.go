package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCatalog(t *testing.T) {
	broker, _, ctx := setupTest()

	services, err := broker.Services(ctx)

	assert.NoError(t, err)
	assert.NotZero(t, len(services), "Expected a non-zero amount of services")

	for _, service := range services {
		assert.NotZerof(t, len(service.Plans), "Expected a non-zero amount of plans for service %s", service.Name)
	}
}

func TestWhitelist(t *testing.T) {
	_, _, ctx := setupTest()

	logger := zap.S()
	whitelist := Whitelist{}
	whitelist["AWS"] = []string{"M10"}
	broker := New(logger, nil, "", whitelist, BasicAuth)
	services, err := broker.Services(ctx)

	require.Len(t, services, 1)
	require.Len(t, services[0].Plans, 1)
	require.NoError(t, err)
}
