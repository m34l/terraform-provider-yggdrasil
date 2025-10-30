package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tfTypes "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &YggdrasilProvider{}

func New() provider.Provider {
	return &YggdrasilProvider{}
}

type YggdrasilProvider struct{}

type YggdrasilProviderModel struct {
	Endpoint           tfTypes.String `tfsdk:"endpoint"`
	Token              tfTypes.String `tfsdk:"token"`
	NamespaceDefault   tfTypes.String `tfsdk:"namespace_default"`
	InsecureSkipVerify tfTypes.Bool   `tfsdk:"insecure_skip_verify"`
	CACertPath         tfTypes.String `tfsdk:"ca_cert_path"`
	ClientCertPath     tfTypes.String `tfsdk:"client_cert_path"`
	ClientKeyPath      tfTypes.String `tfsdk:"client_key_path"`
}

func (p *YggdrasilProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "yggdrasil"
}

func (p *YggdrasilProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "API endpoint URL. Can also be set via YGG_ENDPOINT environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API authentication token. Can also be set via YGG_TOKEN environment variable.",
			},
			"namespace_default": schema.StringAttribute{
				Optional:    true,
				Description: "Default namespace for secrets.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification (development only).",
			},
			"ca_cert_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to CA certificate file.",
			},
			"client_cert_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to client certificate file for mTLS.",
			},
			"client_key_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to client key file for mTLS.",
			},
		},
	}
}

func (p *YggdrasilProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data YggdrasilProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := getStringValue(data.Endpoint, os.Getenv("YGG_ENDPOINT"))
	token := getStringValue(data.Token, os.Getenv("YGG_TOKEN"))

	if endpoint == "" {
		resp.Diagnostics.AddError("Missing endpoint", "Endpoint must be set via config or YGG_ENDPOINT")
		return
	}
	if token == "" {
		resp.Diagnostics.AddError("Missing token", "Token must be set via config or YGG_TOKEN")
		return
	}

	cfg := Config{
		Endpoint:           endpoint,
		Token:              token,
		NamespaceDefault:   data.NamespaceDefault.ValueString(),
		InsecureSkipVerify: data.InsecureSkipVerify.ValueBool(),
		CACertPath:         data.CACertPath.ValueString(),
		ClientCertPath:     data.ClientCertPath.ValueString(),
		ClientKeyPath:      data.ClientKeyPath.ValueString(),
		APIVersion:         "v2", // hardcoded to v2
	}

	client, err := newClient(cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
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

func getStringValue(tfVal tfTypes.String, envVal string) string {
	if !tfVal.IsNull() && tfVal.ValueString() != "" {
		return tfVal.ValueString()
	}
	return envVal
}
