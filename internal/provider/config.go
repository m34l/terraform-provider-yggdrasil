package provider

type Config struct {
	Endpoint           string
	Token              string
	NamespaceDefault   string
	InsecureSkipVerify bool
	CACertPath         string
	ClientCertPath     string
	ClientKeyPath      string
	APIVersion         string // e.g. "v2"
}
