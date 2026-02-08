package dbgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ZeroValue(t *testing.T) {
	var cfg Config

	assert.Empty(t, cfg.PrimaryDSN)
	assert.Empty(t, cfg.ReplicasDSN)
	assert.False(t, cfg.EnableTracing)
	assert.Empty(t, cfg.TracingServiceName)
	assert.Nil(t, cfg.TracingAnalyticsRate)
	assert.Nil(t, cfg.TracingErrorCheck)
}

func TestConfig_WithFields(t *testing.T) {
	errCheck := func(err error) bool { return err != nil }
	rate := 0.5
	cfg := Config{
		PrimaryDSN:          "host=localhost dbname=test",
		ReplicasDSN:         []string{"host=replica1", "host=replica2"},
		EnableTracing:       true,
		TracingServiceName:  "my-service",
		TracingAnalyticsRate: &rate,
		TracingErrorCheck:   errCheck,
	}

	assert.Equal(t, "host=localhost dbname=test", cfg.PrimaryDSN)
	assert.Len(t, cfg.ReplicasDSN, 2)
	assert.Equal(t, "host=replica1", cfg.ReplicasDSN[0])
	assert.Equal(t, "host=replica2", cfg.ReplicasDSN[1])
	assert.True(t, cfg.EnableTracing)
	assert.Equal(t, "my-service", cfg.TracingServiceName)
	assert.NotNil(t, cfg.TracingAnalyticsRate)
	assert.Equal(t, 0.5, *cfg.TracingAnalyticsRate)
	assert.NotNil(t, cfg.TracingErrorCheck)
}

func TestConfig_Validate_EmptyPrimaryDSN_ReturnsError(t *testing.T) {
	cfg := Config{}
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestConfig_Validate_Valid_ReturnsNil(t *testing.T) {
	cfg := Config{PrimaryDSN: "host=localhost dbname=test"}
	err := cfg.Validate()
	assert.NoError(t, err)
}
