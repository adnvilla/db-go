package dbgo

type Config struct {
	PrimaryDSN  string
	ReplicasDSN []string
	// Datadog Tracing configuration
	EnableTracing        bool
	TracingServiceName   string
	TracingAnalyticsRate float64
	TracingErrorCheck    func(error) bool
}
