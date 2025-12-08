// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TokenResource{}
var _ resource.ResourceWithImportState = &TokenResource{}

func NewTokenResource() resource.Resource {
	return &TokenResource{}
}

// TokenResource defines the resource implementation.
type TokenResource struct {
	client *Client
}

// TokenResourceModel describes the resource data model.
type TokenResourceModel struct {
	AllowedUrls types.Set    `tfsdk:"allowed_urls"`
	Id          types.String `tfsdk:"id"`
	Note        types.String `tfsdk:"note"`
	Scopes      types.Set    `tfsdk:"scopes"`
	Token       types.String `tfsdk:"token"`
	Username    types.String `tfsdk:"username"`
}

type tokenCreateBody struct {
	AllowedUrls []string `json:"allowedUrls,omitempty"`
	Id          *string  `json:"id,omitempty"`
	Note        string   `json:"note"`
	Scopes      []string `json:"scopes"`
	Token       *string  `json:"token,omitempty"`
}

func (r *TokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *TokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Token resource",

		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the account for which to list scopes.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A description for the token.",
				Required:            true,
			},
			"scopes": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Specify the scopes that the new token will have. The authorizing token needs to have the same scopes as, or more scopes than, the new token you are creating.",
				Required:            true,
			},
			"allowed_urls": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "URLs that this token is allowed to work with.",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Token value",
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Token identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *TokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TokenResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Error", "Provider client is not configured")
		return
	}

	urls := make([]string, 0, len(data.AllowedUrls.Elements()))
	data.AllowedUrls.ElementsAs(ctx, &urls, false)

	scopes := make([]string, 0, len(data.Scopes.Elements()))
	data.Scopes.ElementsAs(ctx, &scopes, false)

	createBody := tokenCreateBody{
		Note:        data.Note.ValueString(),
		Scopes:      scopes,
		AllowedUrls: urls,
	}

	bytedata, err := json.Marshal(createBody)

	if err != nil {
		resp.Diagnostics.AddError("Parsing Error", fmt.Sprintf("Unable to parse resquest, got error: %s", err))
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	httpReq, err := r.client.Post(fmt.Sprintf("tokens/v2/%s", data.Username.ValueString()), bytes.NewBuffer(bytedata))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create token, got error: %s", err))
		return
	}
	defer func() {
		_ = httpReq.Body.Close()
	}()

	body, readerr := io.ReadAll(httpReq.Body)
	if readerr != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to create token, got error: %s", readerr))
		return
	}

	var token tokenCreateBody

	decodeerr := json.Unmarshal(body, &token)
	if decodeerr != nil {
		resp.Diagnostics.AddError("Unmarshall Error", fmt.Sprintf("Unable to create token, got error: %s", decodeerr))
		return
	}

	// For the purposes of this token code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue(fmt.Sprintf("%s:%s", *token.Id, data.Username.ValueString()))
	data.Token = types.StringValue(*token.Token)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TokenResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, userName, _ := tokenId(data.Id.ValueString())

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	httpReq, err := r.client.Get(fmt.Sprintf("tokens/v2/%s", userName))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read token, got error: %s", err))
		return
	}
	defer func() {
		_ = httpReq.Body.Close()
	}()

	var tokens []*tokenCreateBody
	body, readerr := io.ReadAll(httpReq.Body)
	if readerr != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to read token, got error: %s", readerr))
		return
	}

	decodeerr := json.Unmarshal(body, &tokens)
	if decodeerr != nil {
		resp.Diagnostics.AddError("Unmarshall Error", fmt.Sprintf("Unable to read token, got error: %s", decodeerr))
		return
	}

	var token *tokenCreateBody

	for _, tk := range tokens {
		if *tk.Id == id {
			token = tk
			break
		}
	}

	data.Note = types.StringValue(token.Note)
	data.Username = types.StringValue(userName)
	data.Token = types.StringPointerValue(token.Token)

	if len(token.AllowedUrls) > 0 {
		allowedUrls, _ := types.SetValueFrom(ctx, types.StringType, token.AllowedUrls)
		data.AllowedUrls = allowedUrls
	}

	scopes, _ := types.SetValueFrom(ctx, types.StringType, token.Scopes)
	data.Scopes = scopes

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TokenResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	urls := make([]string, 0, len(data.AllowedUrls.Elements()))
	data.AllowedUrls.ElementsAs(ctx, &urls, false)

	scopes := make([]string, 0, len(data.Scopes.Elements()))
	data.Scopes.ElementsAs(ctx, &scopes, false)

	createBody := tokenCreateBody{
		Note:        data.Note.ValueString(),
		Scopes:      scopes,
		AllowedUrls: urls,
	}

	bytedata, err := json.Marshal(createBody)

	if err != nil {
		resp.Diagnostics.AddError("Parsing Error", fmt.Sprintf("Unable to parse resquest, got error: %s", err))
		return
	}

	id, userName, _ := tokenId(data.Id.ValueString())

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	updateResp, err := r.client.Patch((fmt.Sprintf("tokens/v2/%s/%s", userName, id)), bytes.NewBuffer(bytedata))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update token, got error: %s", err))
		return
	}
	if updateResp != nil {
		defer func() {
			_ = updateResp.Body.Close()
		}()
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TokenResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, userName, _ := tokenId(data.Id.ValueString())

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	deleteResp, err := r.client.Delete(fmt.Sprintf("tokens/v2/%s/%s", userName, id))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete token, got error: %s", err))
		return
	}
	if deleteResp != nil {
		defer func() {
			_ = deleteResp.Body.Close()
		}()
	}
}

func (r *TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func tokenId(id string) (string, string, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected TOKEN-ID:USERNAME", id)
	}

	return parts[0], parts[1], nil
}
