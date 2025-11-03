package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsSchema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	tfTypes "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SecretDataSource{}

func NewSecretDataSource() datasource.DataSource {
	return &SecretDataSource{}
}

type SecretDataSource struct {
	client *APIClient
}

type SecretDataModel struct {
	ID        tfTypes.String `tfsdk:"id"`
	Namespace tfTypes.String `tfsdk:"namespace"`
	Key       tfTypes.String `tfsdk:"key"`
	Tag       tfTypes.String `tfsdk:"tag"` // Single tag (required)
	Value     tfTypes.String `tfsdk:"value"`
	Tags      tfTypes.Map    `tfsdk:"tags"`
	Version   tfTypes.Int64  `tfsdk:"version"`
	UpdatedAt tfTypes.String `tfsdk:"updated_at"`
}

func (d *SecretDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "yggdrasil_secret"
}

func (d *SecretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsSchema.Schema{
		Description: "Fetch a secret from Yggdrasil configuration service",
		Attributes: map[string]dsSchema.Attribute{
			"namespace": dsSchema.StringAttribute{
				Required:    true,
				Description: "The namespace where the secret is stored",
			},
			"key": dsSchema.StringAttribute{
				Required:    true,
				Description: "The key name of the secret",
			},
			"tag": dsSchema.StringAttribute{
				Required:    true,
				Description: "Specific tag to fetch value from (e.g. 'production', 'staging'). No fallback.",
			},
			"value": dsSchema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The secret value from the specified tag",
			},
			"tags": dsSchema.MapAttribute{
				ElementType: tfTypes.StringType,
				Computed:    true,
			},
			"version": dsSchema.Int64Attribute{
				Computed:    true,
				Description: "Version of the configuration",
			},
			"updated_at": dsSchema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
			},
			"id": dsSchema.StringAttribute{
				Computed:    true,
				Description: "Identifier in format namespace/key@tag",
			},
		},
	}
}

func (d *SecretDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(*APIClient)
}

func (d *SecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SecretDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns := data.Namespace.ValueString()
	key := data.Key.ValueString()
	tag := data.Tag.ValueString()

	out, err := d.client.GetSecret(ns, key, tag)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read secret",
			fmt.Sprintf("Could not read secret %s/%s@%s: %s", ns, key, tag, err.Error()),
		)
		return
	}

	if out == nil {
		resp.Diagnostics.AddError(
			"Secret not found",
			fmt.Sprintf("Secret with key '%s' does not exist in namespace '%s' tag '%s'", key, ns, tag),
		)
		return
	}

	// Set all computed values
	data.ID = tfTypes.StringValue(fmt.Sprintf("%s/%s@%s", out.Namespace, out.Key, tag))
	data.Value = tfTypes.StringValue(out.Value)
	data.Version = tfTypes.Int64Value(int64(out.Version))
	data.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)

	if out.Tags != nil && len(out.Tags) > 0 {
		elems := map[string]attr.Value{}
		for k, v := range out.Tags {
			elems[k] = tfTypes.StringValue(v)
		}
		data.Tags = tfTypes.MapValueMust(tfTypes.StringType, elems)
	} else {
		data.Tags = tfTypes.MapNull(tfTypes.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
