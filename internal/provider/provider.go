// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure MapBoxProvider satisfies various provider interfaces.
var _ provider.Provider = &MapBoxProvider{}

// MapBoxProvider defines the provider implementation.
type MapBoxProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// MapBoxProviderModel describes the provider data model.
type MapBoxProviderModel struct {
	AccessToken types.String `tfsdk:"access_token"`
}

func (p *MapBoxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "mapbox"
	resp.Version = p.version
}

func (p *MapBoxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access token to authenticate to mapbox with",
				Optional:            true,
			},
		},
	}
}

func (p *MapBoxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	accessToken := os.Getenv("MAPBOX_ACCESS_TOKEN")

	var data MapBoxProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.AccessToken.IsNull() { /* ... */ }

	// Example client configuration for data sources and resources
	client := &Client{
		HTTPClient: http.DefaultClient,
	}

	if data.AccessToken.ValueString() != "" {
		accessToken = data.AccessToken.ValueString()
	}

	if accessToken == "" {
		resp.Diagnostics.AddError(
			"Missing Access Token Configuration",
			"While configuring the provider, the API token was not found in "+
				"the MAPBOX_ACCESS_TOKEN environment variable or provider "+
				"configuration block access_token attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	client.AccessToken = &accessToken
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *MapBoxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTokenResource,
	}
}

func (p *MapBoxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MapBoxProvider{
			version: version,
		}
	}
}
