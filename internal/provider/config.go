package provider

type Config struct {
	Endpoint           string
	Token              string // For read operations (data sources)
	CredKey            string // For write operations (resources) - key-secret pair
	CredSecret         string // For write operations (resources) - key-secret pair
	NamespaceDefault   string
	InsecureSkipVerify bool
	CACertPath         string
	ClientCertPath     string
	ClientKeyPath      string

	// API
	APIVersion string // "v2" default

	// Auth
	AuthScheme string // "bearer" | "token" | "x-api-key"
	AuthHeader string // override header name (e.g. "X-API-KEY")
}
