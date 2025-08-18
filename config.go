package dbgo

type Config struct {
	PrimaryDSN  string
	ReplicasDSN []string
}
