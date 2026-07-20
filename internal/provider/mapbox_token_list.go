// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ list.ListResource = &TokenListResource{}
var _ list.ListResourceWithConfigure = &TokenListResource{}

func NewTokenListResource() list.ListResource {
	return &TokenListResource{}
}

// TokenListResource lists mapbox_token managed resources.
type TokenListResource struct {
	client *Client
}

// TokenListConfigModel describes the list block configuration data model.
type TokenListConfigModel struct {
	Username types.String `tfsdk:"username"`
}

func (r *TokenListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	// Must match the managed resource type name being listed.
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *TokenListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List the access tokens owned by a mapbox account.",
		Attributes: map[string]listschema.Attribute{
			"username": listschema.StringAttribute{
				MarkdownDescription: "The username of the account whose tokens should be listed.",
				Required:            true,
			},
		},
	}
}

func (r *TokenListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected List Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *TokenListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var config TokenListConfigModel

	if diags := req.Config.Get(ctx, &config); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	if r.client == nil {
		result := req.NewListResult(ctx)
		result.Diagnostics.AddError("Client Error", "Provider client is not configured")
		stream.Results = slices.Values([]list.ListResult{result})
		return
	}

	userName := config.Username.ValueString()

	httpReq, err := r.client.Get(fmt.Sprintf("tokens/v2/%s", userName))
	if err != nil {
		result := req.NewListResult(ctx)
		result.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list tokens, got error: %s", err))
		stream.Results = slices.Values([]list.ListResult{result})
		return
	}
	defer func() {
		_ = httpReq.Body.Close()
	}()

	body, readerr := io.ReadAll(httpReq.Body)
	if readerr != nil {
		result := req.NewListResult(ctx)
		result.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to list tokens, got error: %s", readerr))
		stream.Results = slices.Values([]list.ListResult{result})
		return
	}

	var tokens []*tokenCreateBody
	if decodeerr := json.Unmarshal(body, &tokens); decodeerr != nil {
		result := req.NewListResult(ctx)
		result.Diagnostics.AddError("Unmarshall Error", fmt.Sprintf("Unable to list tokens, got error: %s", decodeerr))
		stream.Results = slices.Values([]list.ListResult{result})
		return
	}

	results := make([]list.ListResult, 0, len(tokens))

	for _, token := range tokens {
		result := req.NewListResult(ctx)

		if token.Id == nil {
			result.Diagnostics.AddError("Unexpected API Response", "Mapbox returned a token without an id.")
			results = append(results, result)
			continue
		}

		result.DisplayName = fmt.Sprintf("%s (%s)", token.Note, *token.Id)

		result.Diagnostics.Append(result.Identity.Set(ctx, TokenResourceIdentityModel{
			TokenId:  types.StringValue(*token.Id),
			Username: types.StringValue(userName),
		})...)

		if req.IncludeResource {
			model := TokenResourceModel{
				Id:       types.StringValue(fmt.Sprintf("%s:%s", *token.Id, userName)),
				Username: types.StringValue(userName),
				Note:     types.StringValue(token.Note),
				Token:    types.StringPointerValue(token.Token),
			}

			scopes, scopeDiags := types.SetValueFrom(ctx, types.StringType, token.Scopes)
			result.Diagnostics.Append(scopeDiags...)
			model.Scopes = scopes

			if len(token.AllowedUrls) > 0 {
				allowedUrls, urlDiags := types.SetValueFrom(ctx, types.StringType, token.AllowedUrls)
				result.Diagnostics.Append(urlDiags...)
				model.AllowedUrls = allowedUrls
			} else {
				model.AllowedUrls = types.SetNull(types.StringType)
			}

			result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
		}

		results = append(results, result)
	}

	stream.Results = slices.Values(results)
}
