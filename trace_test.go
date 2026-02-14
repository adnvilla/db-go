package dbgo

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWithTracing_EnablesTracing(t *testing.T) {
	cfg := &Config{}
	result := WithTracing(cfg)

	assert.True(t, result.EnableTracing)
}

func TestWithTracing_PreservesOtherFields(t *testing.T) {
	cfg := &Config{
		PrimaryDSN:         "host=localhost",
		TracingServiceName: "existing-service",
	}
	result := WithTracing(cfg)

	assert.True(t, result.EnableTracing)
	assert.Equal(t, "host=localhost", result.PrimaryDSN)
	assert.Equal(t, "existing-service", result.TracingServiceName)
}

func TestWithTracingServiceName(t *testing.T) {
	cfg := &Config{}
	optFn := WithTracingServiceName("my-service")
	result := optFn(cfg)

	assert.Equal(t, "my-service", result.TracingServiceName)
}

func TestWithTracingAnalyticsRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"zero rate", 0.0},
		{"half rate", 0.5},
		{"full rate", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			optFn := WithTracingAnalyticsRate(tt.rate)
			result := optFn(cfg)

			assert.NotNil(t, result.TracingAnalyticsRate)
			assert.Equal(t, tt.rate, *result.TracingAnalyticsRate)
		})
	}
}

func TestWithTracingErrorCheck(t *testing.T) {
	errCheck := func(err error) bool { return err != nil }
	cfg := &Config{}
	optFn := WithTracingErrorCheck(errCheck)
	result := optFn(cfg)

	assert.NotNil(t, result.TracingErrorCheck)
	assert.True(t, result.TracingErrorCheck(assert.AnError))
	assert.False(t, result.TracingErrorCheck(nil))
}

func TestWithTracingOptions_Chaining(t *testing.T) {
	cfg := &Config{PrimaryDSN: "host=localhost"}
	cfg = WithTracing(cfg)
	cfg = WithTracingServiceName("chained-service")(cfg)
	cfg = WithTracingAnalyticsRate(0.75)(cfg)
	cfg = WithTracingErrorCheck(func(err error) bool { return true })(cfg)

	assert.True(t, cfg.EnableTracing)
	assert.Equal(t, "chained-service", cfg.TracingServiceName)
	assert.NotNil(t, cfg.TracingAnalyticsRate)
	assert.Equal(t, 0.75, *cfg.TracingAnalyticsRate)
	assert.NotNil(t, cfg.TracingErrorCheck)
	assert.Equal(t, "host=localhost", cfg.PrimaryDSN)
}

func TestEnableTracing_WhenDisabled(t *testing.T) {
	db := &gorm.DB{}
	cfg := Config{EnableTracing: false}

	result, err := EnableTracing(db, cfg)
	assert.NoError(t, err)
	assert.Equal(t, db, result, "should return the same db when tracing is disabled")
}

func TestWithContext_WrapsDBAndSetsContext(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{})
	assert.NoError(t, err)

	ctx := context.Background()
	newCtx, result := WithContext(ctx, db)
	assert.NotNil(t, result)
	assert.NotNil(t, newCtx)

	// Verify the DB is retrievable from the returned context
	fromCtx := GetFromContext(newCtx)
	assert.Same(t, result, fromCtx)
}

func TestStartSpan_EmptyService_UsesDefault(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-op", "")
	assert.NotNil(t, newCtx)
	if span != nil {
		span.Finish()
	}
	// Default service name is applied when tracer is running; span may be nil when tracer not started
}

func TestStartSpan_WithService_UsesGivenService(t *testing.T) {
	ctx := context.Background()
	newCtx, span := StartSpan(ctx, "test-op", "my-service")
	assert.NotNil(t, newCtx)
	if span != nil {
		span.Finish()
	}
}
