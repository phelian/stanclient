package stanclient

// Config type for stanclient
type Config struct {
	Enabled          bool   `json:"enabled"`
	ConnectRetry     Retry  `json:"connect_retry"`
	ReconnectRetry   Retry  `json:"reconnect_retry"`
	ClientID         string `json:"client_id"`
	ClusterID        string `json:"cluster_id"`
	NatsStreamingURL string `json:"nats_streaming_url"`
}

// Retry configuration
type Retry struct {
	Attempts uint  `json:"attempts"`
	Delay    int64 `json:"delay"` // Seconds
}
