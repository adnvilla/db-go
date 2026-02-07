package dbgo

import (
	"context"

	gormtrace "github.com/DataDog/dd-trace-go/contrib/gorm.io/gorm.v1/v2"
	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"gorm.io/gorm"
)

// WithTracing enables Datadog tracing for GORM operations.
// Use this function to enable tracing in your database configuration.
// Example:
//
//	config := dbgo.Config{PrimaryDSN: "..."}
//	config = *dbgo.WithTracing(&config)
func WithTracing(cfg *Config) *Config {
	cfg.EnableTracing = true
	return cfg
}

// WithTracingServiceName sets the service name for Datadog tracing.
// The service name will appear in your Datadog APM dashboard.
// Example:
//
//	config := dbgo.Config{PrimaryDSN: "..."}
//	config = *dbgo.WithTracing(&config)
//	config = *dbgo.WithTracingServiceName("my-db-service")(&config)
func WithTracingServiceName(serviceName string) func(*Config) *Config {
	return func(cfg *Config) *Config {
		cfg.TracingServiceName = serviceName
		return cfg
	}
}

// WithTracingAnalyticsRate sets the analytics rate for Datadog tracing.
// This determines what percentage of traces will be analyzed.
// Values should be between 0.0 and 1.0, where 1.0 means 100% of traces are analyzed.
// Example:
//
//	config := dbgo.Config{PrimaryDSN: "..."}
//	config = *dbgo.WithTracing(&config)
//	config = *dbgo.WithTracingAnalyticsRate(1.0)(&config)
func WithTracingAnalyticsRate(rate float64) func(*Config) *Config {
	return func(cfg *Config) *Config {
		cfg.TracingAnalyticsRate = &rate
		return cfg
	}
}

// WithTracingErrorCheck sets a custom error check function for Datadog tracing.
// This allows you to control which errors are reported to Datadog.
// Example:
//
//	config := dbgo.Config{PrimaryDSN: "..."}
//	config = *dbgo.WithTracing(&config)
//	config = *dbgo.WithTracingErrorCheck(func(err error) bool {
//	    // Only report non-nil errors
//	    return err != nil
//	})(&config)
func WithTracingErrorCheck(errCheck func(error) bool) func(*Config) *Config {
	return func(cfg *Config) *Config {
		cfg.TracingErrorCheck = errCheck
		return cfg
	}
}

// EnableTracing applies Datadog tracing to a GORM database connection.
// This function is called internally by getConnection when tracing is enabled.
// You generally don't need to call this function directly.
func EnableTracing(db *gorm.DB, cfg Config) (*gorm.DB, error) {
	if !cfg.EnableTracing {
		return db, nil
	}

	var opts []gormtrace.Option

	if cfg.TracingServiceName != "" {
		opts = append(opts, gormtrace.WithService(cfg.TracingServiceName))
	}

	if cfg.TracingAnalyticsRate != nil {
		opts = append(opts, gormtrace.WithAnalyticsRate(*cfg.TracingAnalyticsRate))
	}

	if cfg.TracingErrorCheck != nil {
		opts = append(opts, gormtrace.WithErrorCheck(cfg.TracingErrorCheck))
	}

	plugin := gormtrace.NewTracePlugin(opts...)
	if err := db.Use(plugin); err != nil {
		return nil, err
	}

	return db, nil
}

// WithContext wraps the GORM database connection with a context that might contain
// a Datadog span, allowing to trace operations within transaction blocks.
// Use this to propagate tracing context through your application.
// Example:
//
//	span, ctx := tracer.StartSpanFromContext(context.Background(), "my-operation")
//	defer span.Finish()
//	db := dbgo.WithContext(ctx, dbConn.Instance)
func WithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.WithContext(ctx)
}

// StartSpan creates a new Datadog span from the given context.
// This can be used to create a parent span before executing database operations.
// Example:
//
//	ctx, span := dbgo.StartSpan(context.Background(), "database-operations", "my-service")
//	defer span.Finish()
//	db := dbgo.WithContext(ctx, dbConn.Instance)
func StartSpan(ctx context.Context, name, service string) (context.Context, *tracer.Span) {
	span, ctx := tracer.StartSpanFromContext(ctx, name,
		tracer.ServiceName(service),
	)
	return ctx, span
}
