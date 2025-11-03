package provider

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	pfTypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider implementation
var _ provider.Provider = &YggdrasilProvider{}

func New() provider.Provider {
	return &YggdrasilProvider{}
}

type YggdrasilProvider struct{}

type yggdrasilProviderModel struct {
	Endpoint           pfTypes.String `tfsdk:"endpoint"`
	Token              pfTypes.String `tfsdk:"token"`
	CredKey            pfTypes.String `tfsdk:"cred_key"`
	CredSecret         pfTypes.String `tfsdk:"cred_secret"`
	NamespaceDefault   pfTypes.String `tfsdk:"namespace_default"`
	InsecureSkipVerify pfTypes.Bool   `tfsdk:"insecure_skip_verify"`
	CACertPath         pfTypes.String `tfsdk:"ca_cert_path"`
	ClientCertPath     pfTypes.String `tfsdk:"client_cert_path"`
	ClientKeyPath      pfTypes.String `tfsdk:"client_key_path"`
	APIVersion         pfTypes.String `tfsdk:"api_version"`
	AuthScheme         pfTypes.String `tfsdk:"auth_scheme"`
	AuthHeader         pfTypes.String `tfsdk:"auth_header"`
}

func (p *YggdrasilProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "yggdrasil"
}

func (p *YggdrasilProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL to Yggdrasil API gateway, e.g. https://yggdrasil.api.example.com",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Token for read operations (data sources). Can also be set via YGG_TOKEN.",
			},
			"cred_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Credential key for write operations (resources). Can also be set via YGG_CRED_KEY.",
			},
			"cred_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Credential secret for write operations (resources). Can also be set via YGG_CRED_SECRET.",
			},
			"namespace_default": schema.StringAttribute{
				Optional:    true,
				Description: "Default namespace if resource doesn't specify one.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS cert verification (NOT recommended in production).",
			},
			"ca_cert_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to custom CA certificate (PEM).",
			},
			"client_cert_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to mTLS client certificate (PEM).",
			},
			"client_key_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to mTLS client private key (PEM).",
			},
			"api_version": schema.StringAttribute{
				Optional:    true,
				Description: "API version: v1 or v2. Default v2.",
			},
			"auth_scheme": schema.StringAttribute{
				Optional:    true,
				Description: "Auth scheme: bearer|token|x-api-key. Default bearer. Ignored if auth_header is set.",
			},
			"auth_header": schema.StringAttribute{
				Optional:    true,
				Description: "Override header name (e.g., X-API-KEY). If set, scheme is ignored and token is sent as-is.",
			},
		},
	}
}

func (p *YggdrasilProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data yggdrasilProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := Config{
		Endpoint:           firstNonEmpty(data.Endpoint.ValueString(), os.Getenv("YGG_ENDPOINT")),
		Token:              firstNonEmpty(data.Token.ValueString(), os.Getenv("YGG_TOKEN")),
		CredKey:            firstNonEmpty(data.CredKey.ValueString(), os.Getenv("YGG_CRED_KEY")),
		CredSecret:         firstNonEmpty(data.CredSecret.ValueString(), os.Getenv("YGG_CRED_SECRET")),
		NamespaceDefault:   firstNonEmpty(data.NamespaceDefault.ValueString(), os.Getenv("YGG_NAMESPACE")),
		InsecureSkipVerify: data.InsecureSkipVerify.ValueBool(),
		CACertPath:         data.CACertPath.ValueString(),
		ClientCertPath:     data.ClientCertPath.ValueString(),
		ClientKeyPath:      data.ClientKeyPath.ValueString(),
		APIVersion:         firstNonEmpty(data.APIVersion.ValueString(), os.Getenv("YGG_API_VERSION"), "v2"),
		AuthScheme:         strings.ToLower(firstNonEmpty(data.AuthScheme.ValueString(), os.Getenv("YGG_AUTH_SCHEME"), "bearer")),
		AuthHeader:         firstNonEmpty(data.AuthHeader.ValueString(), os.Getenv("YGG_AUTH_HEADER")),
	}

	// DEBUG: Log credential status (without exposing actual values)
	log.Printf("[DEBUG] Provider Config Loaded:")
	log.Printf("[DEBUG]   Endpoint: %s", cfg.Endpoint)
	log.Printf("[DEBUG]   Token: %v (length: %d)", cfg.Token != "", len(cfg.Token))
	log.Printf("[DEBUG]   CredKey: %v (length: %d)", cfg.CredKey != "", len(cfg.CredKey))
	log.Printf("[DEBUG]   CredSecret: %v (length: %d)", cfg.CredSecret != "", len(cfg.CredSecret))
	log.Printf("[DEBUG]   API Version: %s", cfg.APIVersion)

	c, err := newClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to init client", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *YggdrasilProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSecretResource,
	}
}

func (p *YggdrasilProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSecretDataSource,
	}
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
