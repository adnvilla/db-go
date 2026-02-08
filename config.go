package dbgo

// Config holds the settings for the database connection and optional features.
type Config struct {
	// PrimaryDSN is the data source name for the primary (read-write) PostgreSQL instance. Required.
	PrimaryDSN string

	// ReplicasDSN is the list of DSNs for read-only replicas. Queries that do not use dbresolver.Write
	// may be executed against one of these replicas (policy: random). Leave nil or empty for no replicas.
	ReplicasDSN []string

	// EnableTracing turns on Datadog APM tracing for GORM operations when true.
	EnableTracing bool

	// TracingServiceName is the service name shown in Datadog. If empty, the tracer default is used.
	// See DefaultTracingServiceName for the default used by dbgo when not set.
	TracingServiceName string

	// TracingAnalyticsRate sets the fraction of traces sent to analytics (0.0 to 1.0). Nil uses tracer default.
	TracingAnalyticsRate *float64

	// TracingErrorCheck is the function used to decide if an error is reported as an error span in Datadog.
	// If nil, the tracing plugin's default behavior is used.
	TracingErrorCheck func(error) bool
}

// Validate checks that Config has required fields. Returns an error suitable for DBConn.Error when invalid.
func (c Config) Validate() error {
	if c.PrimaryDSN == "" {
		return ErrInvalidConfig
	}
	return nil
}
