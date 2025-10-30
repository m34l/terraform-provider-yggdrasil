package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	stringplanmodifier "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	tfTypes "github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

type SecretResource struct {
	client *APIClient
}

type SecretResourceModel struct {
	ID        tfTypes.String `tfsdk:"id"`
	Namespace tfTypes.String `tfsdk:"namespace"`
	Key       tfTypes.String `tfsdk:"key"`
	Value     tfTypes.String `tfsdk:"value"` // Sensitive
	Tags      tfTypes.Map    `tfsdk:"tags"`
	Version   tfTypes.Int64  `tfsdk:"version"`
	UpdatedAt tfTypes.String `tfsdk:"updated_at"`
}

func (r *SecretResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "yggdrasil_secret"
}

func (r *SecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resSchema.Schema{
		Attributes: map[string]resSchema.Attribute{
			"id": resSchema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace": resSchema.StringAttribute{
				Required: true,
			},
			"key": resSchema.StringAttribute{
				Required: true,
			},
			"value": resSchema.StringAttribute{
				Required:  true,
				Sensitive: true,
			},
			"tags": resSchema.MapAttribute{
				ElementType: tfTypes.StringType,
				Optional:    true,
			},
			"version": resSchema.Int64Attribute{
				Computed: true,
			},
			"updated_at": resSchema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *SecretResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*APIClient)
}

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := SecretPayload{
		Namespace: plan.Namespace.ValueString(),
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		Tags:      mapFromTF(ctx, plan.Tags),
	}

	out, err := r.client.UpsertSecret(payload)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}

	state := plan
	state.ID = tfTypes.StringValue(fmt.Sprintf("%s/%s", out.Namespace, out.Key))
	state.Version = tfTypes.Int64Value(int64(out.Version))
	state.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns := state.Namespace.ValueString()
	key := state.Key.ValueString()
	out, err := r.client.GetSecret(ns, key)
	if err != nil {
		resp.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	if out == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.Version = tfTypes.Int64Value(int64(out.Version))
	state.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)
	// Jangan set ulang Value dari remote bila API tidak mengembalikan (atau redaksi)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecretResourceModel
	var state SecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := SecretPayload{
		Namespace: plan.Namespace.ValueString(),
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		Tags:      mapFromTF(ctx, plan.Tags),
	}
	out, err := r.client.UpsertSecret(payload)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}
	state = plan
	state.ID = tfTypes.StringValue(fmt.Sprintf("%s/%s", out.Namespace, out.Key))
	state.Version = tfTypes.Int64Value(int64(out.Version))
	state.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSecret(state.Namespace.ValueString(), state.Key.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete failed", err.Error())
	}
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// import_id format: "namespace/key"
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	// Split manual
	var ns, key string
	for i, ch := range req.ID {
		if ch == '/' {
			ns = req.ID[:i]
			key = req.ID[i+1:]
			break
		}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("namespace"), ns)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), key)...)
}
func mapFromTF(ctx context.Context, m tfTypes.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	out := make(map[string]string)
	// false = jangan set unknown ke null; sesuaikan kebutuhan
	_ = m.ElementsAs(ctx, &out, false)
	return out
}
