// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &TokenEphemeralResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &TokenEphemeralResource{}

func NewTokenEphemeralResource() ephemeral.EphemeralResource {
	return &TokenEphemeralResource{}
}

// TokenEphemeralResource reads the value of an existing public token without
// persisting it in plan or state.
type TokenEphemeralResource struct {
	client *Client
}

type TokenEphemeralResourceModel struct {
	Username types.String `tfsdk:"username"`
	TokenId  types.String `tfsdk:"token_id"`
	Token    types.String `tfsdk:"token"`
}

func (r *TokenEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *TokenEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the value of an existing Mapbox token without storing it in plan or state. " +
			"Pair it with a `mapbox_token` resource that sets `store_token = false`. " +
			"Only public tokens are supported: Mapbox returns the value of secret tokens once, at creation.",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The username of the account that owns the token.",
				Required:            true,
			},
			"token_id": schema.StringAttribute{
				MarkdownDescription: "Token identifier.",
				Required:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Token value.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *TokenEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Ephemeral Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *TokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data TokenEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Client Error", "Provider client is not configured")
		return
	}

	tokens, err := r.client.listTokens(data.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list tokens, got error: %s", err))
		return
	}

	id := data.TokenId.ValueString()
	for _, tk := range tokens {
		if tk.Id == nil || *tk.Id != id {
			continue
		}
		if tk.Token == nil || *tk.Token == "" {
			resp.Diagnostics.AddError(
				"Token Value Not Available",
				fmt.Sprintf("Mapbox did not return a value for token %q. Secret tokens only expose their value once, "+
					"at creation, so they cannot be read later. Use a public token or keep store_token = true on the resource.", id),
			)
			return
		}

		data.Token = types.StringPointerValue(tk.Token)
		resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError("Token Not Found", fmt.Sprintf("No token with id %q found for user %q.", id, data.Username.ValueString()))
}
