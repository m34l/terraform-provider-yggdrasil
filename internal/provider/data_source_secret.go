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
	Value     tfTypes.String `tfsdk:"value"` // Sensitive, optional (hanya jika API kembalikan)
	Tags      tfTypes.Map    `tfsdk:"tags"`
	Version   tfTypes.Int64  `tfsdk:"version"`
	UpdatedAt tfTypes.String `tfsdk:"updated_at"`
}

func (d *SecretDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "yggdrasil_secret"
}

func (d *SecretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsSchema.Schema{
		Attributes: map[string]dsSchema.Attribute{
			"namespace": dsSchema.StringAttribute{
				Required: true,
			},
			"key": dsSchema.StringAttribute{
				Required: true,
			},
			"value": dsSchema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"tags": dsSchema.MapAttribute{
				ElementType: tfTypes.StringType,
				Computed:    true,
			},
			"version": dsSchema.Int64Attribute{
				Computed: true,
			},
			"updated_at": dsSchema.StringAttribute{
				Computed: true,
			},
			"id": dsSchema.StringAttribute{
				Computed: true,
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

	out, err := d.client.GetSecret(data.Namespace.ValueString(), data.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Not found", "Secret does not exist")
		return
	}

	data.ID = tfTypes.StringValue(fmt.Sprintf("%s/%s", out.Namespace, out.Key))
	data.Version = tfTypes.Int64Value(int64(out.Version))
	data.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)
	if out.Tags != nil {
		elems := map[string]attr.Value{}
		for k, v := range out.Tags {
			elems[k] = tfTypes.StringValue(v)
		}
		data.Tags = tfTypes.MapValueMust(tfTypes.StringType, elems)
	}
	// Jika API tidak mengembalikan value untuk keamanan, biarkan kosong.
	if out.Value != "" {
		data.Value = tfTypes.StringValue(out.Value)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
