package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Value     tfTypes.String `tfsdk:"value"`
	Tag       tfTypes.String `tfsdk:"tag"`
	Tags      tfTypes.Map    `tfsdk:"tags"`
	Version   tfTypes.Int64  `tfsdk:"version"`
	UpdatedAt tfTypes.String `tfsdk:"updated_at"`
}

func (r *SecretResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "yggdrasil_secret"
}

func (r *SecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resSchema.Schema{
		Description: "Manages a secret in Yggdrasil configuration service",
		Attributes: map[string]resSchema.Attribute{
			"id": resSchema.StringAttribute{
				Computed:    true,
				Description: "Identifier in format namespace/key@tag",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"namespace": resSchema.StringAttribute{
				Required:    true,
				Description: "Namespace where the secret will be stored",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": resSchema.StringAttribute{
				Required:    true,
				Description: "Key name for the secret",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": resSchema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Secret value to store",
			},
			"tag": resSchema.StringAttribute{
				Required:    true,
				Description: "Tag to store this secret under (e.g., 'production', 'staging')",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tags": resSchema.MapAttribute{
				ElementType: tfTypes.StringType,
				Computed:    true,
				Description: "Metadata tags (computed)",
			},
			"version": resSchema.Int64Attribute{
				Computed:    true,
				Description: "Configuration version",
			},
			"updated_at": resSchema.StringAttribute{
				Computed:    true,
				Description: "Last update timestamp",
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

	tag := plan.Tag.ValueString()
	payload := SecretPayload{
		Namespace: plan.Namespace.ValueString(),
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		Tags:      map[string]string{tag: tag},
	}

	out, err := r.client.UpsertSecret(payload)
	if err != nil {
		resp.Diagnostics.AddError("Create failed", err.Error())
		return
	}

	state := plan
	state.ID = tfTypes.StringValue(fmt.Sprintf("%s/%s@%s", out.Namespace, out.Key, tag))
	state.Version = tfTypes.Int64Value(int64(out.Version))
	state.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)

	// FIXED: Set tags with known value
	tagsMap := make(map[string]attr.Value)
	tagsMap[tag] = tfTypes.StringValue(tag)
	state.Tags = tfTypes.MapValueMust(tfTypes.StringType, tagsMap)

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
	tag := state.Tag.ValueString()

	out, err := r.client.GetSecret(ns, key, tag)
	if err != nil {
		resp.Diagnostics.AddError("Read failed", err.Error())
		return
	}
	if out == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Only update computed fields, keep value from state (Terraform best practice)
	state.Version = tfTypes.Int64Value(int64(out.Version))
	state.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)

	// FIXED: Ensure tags is always set
	if state.Tags.IsNull() || state.Tags.IsUnknown() {
		tagsMap := make(map[string]attr.Value)
		tagsMap[tag] = tfTypes.StringValue(tag)
		state.Tags = tfTypes.MapValueMust(tfTypes.StringType, tagsMap)
	}

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

	tag := plan.Tag.ValueString()
	payload := SecretPayload{
		Namespace: plan.Namespace.ValueString(),
		Key:       plan.Key.ValueString(),
		Value:     plan.Value.ValueString(),
		Tags:      map[string]string{tag: tag},
	}

	out, err := r.client.UpsertSecret(payload)
	if err != nil {
		resp.Diagnostics.AddError("Update failed", err.Error())
		return
	}

	state = plan
	state.ID = tfTypes.StringValue(fmt.Sprintf("%s/%s@%s", out.Namespace, out.Key, tag))
	state.Version = tfTypes.Int64Value(int64(out.Version))
	state.UpdatedAt = tfTypes.StringValue(out.UpdatedAt)

	// FIXED: Set tags with known value
	tagsMap := make(map[string]attr.Value)
	tagsMap[tag] = tfTypes.StringValue(tag)
	state.Tags = tfTypes.MapValueMust(tfTypes.StringType, tagsMap)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// NO-OP: We don't actually delete secrets from Yggdrasil
	// This is by design to:
	// 1. Avoid database lock contention issues
	// 2. Prevent accidental deletion of shared configuration keys
	// 3. Allow other tags to continue using the same key
	//
	// The secret will remain in Yggdrasil but removed from Terraform state only.
	// If you need to remove the secret, do it manually via Yggdrasil UI or API.

	ns := state.Namespace.ValueString()
	key := state.Key.ValueString()
	tag := state.Tag.ValueString()

	resp.Diagnostics.AddWarning(
		"Secret not deleted from Yggdrasil",
		fmt.Sprintf(
			"Secret %s/%s@%s has been removed from Terraform state, "+
				"but still exists in Yggdrasil. "+
				"This is intentional to prevent accidental deletion and lock contention. "+
				"To actually remove the secret, use Yggdrasil UI or API directly.",
			ns, key, tag,
		),
	)

	// State is automatically removed by Terraform Framework after this function returns
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// import_id format: "namespace/key@tag"
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	// Parse namespace/key@tag
	var ns, key, tag string
	atIdx := strings.LastIndex(req.ID, "@")
	if atIdx > 0 {
		tag = req.ID[atIdx+1:]
		remaining := req.ID[:atIdx]
		slashIdx := strings.Index(remaining, "/")
		if slashIdx > 0 {
			ns = remaining[:slashIdx]
			key = remaining[slashIdx+1:]
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("namespace"), ns)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tag"), tag)...)
}
