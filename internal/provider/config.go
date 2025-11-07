package provider

type Config struct {
	Endpoint           string
	Token              string
	CredKey            string
	CredSecret         string
	NamespaceDefault   string
	InsecureSkipVerify bool

	// File paths
	CACertPath     string
	ClientCertPath string
	ClientKeyPath  string

	// Base64 encoded certs (NEW)
	CACert     string
	ClientCert string
	ClientKey  string

	// API
	APIVersion string

	// Auth
	AuthScheme string
	AuthHeader string
}
