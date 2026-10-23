// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// readToken runs TokenResource.Read against a fake API returning apiBody and
// returns the resulting state, or nil when the resource was removed.
func readToken(t *testing.T, prior TokenResourceModel, apiBody string) *TokenResourceModel {
	t.Helper()
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(apiBody))
	}))
	defer server.Close()

	token := "pk.test"
	r := &TokenResource{client: &Client{AccessToken: &token, BaseURL: server.URL + "/"}}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	var identityResp resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, prior); diags.HasError() {
		t.Fatalf("set prior state: %v", diags)
	}

	identity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema, Raw: tftypes.NewValue(identityResp.IdentitySchema.Type().TerraformType(ctx), nil)}
	resp := resource.ReadResponse{State: state, Identity: identity}
	r.Read(ctx, resource.ReadRequest{State: state, Identity: identity}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}

	if resp.State.Raw.IsNull() {
		return nil
	}

	var got TokenResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("get state: %v", diags)
	}
	return &got
}

func priorToken(t *testing.T, allowedUrls types.Set) TokenResourceModel {
	t.Helper()
	scopes, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"styles:read"})
	return TokenResourceModel{
		Id:          types.StringValue("abc:user"),
		Username:    types.StringValue("user"),
		Note:        types.StringValue("note"),
		Scopes:      scopes,
		Token:       types.StringValue("sk.stored"),
		AllowedUrls: allowedUrls,
	}
}

func urlSet(t *testing.T, urls ...string) types.Set {
	t.Helper()
	s, diags := types.SetValueFrom(context.Background(), types.StringType, urls)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return s
}

func TestTokenRead_AllowedUrls(t *testing.T) {
	const withoutUrls = `[{"id":"abc","note":"note","scopes":["styles:read"],"usage":"sk"}]`
	const withUrls = `[{"id":"abc","note":"note","scopes":["styles:read"],"allowedUrls":["https://a.example"],"usage":"sk"}]`

	cases := map[string]struct {
		prior   types.Set
		apiBody string
		want    types.Set
	}{
		// 8b1d567: unset in config, API omits it -> stays null (no diff).
		"unset stays null":          {types.SetNull(types.StringType), withoutUrls, types.SetNull(types.StringType)},
		"empty set stays empty":     {urlSet(t), withoutUrls, urlSet(t)},
		"api value is read":         {types.SetNull(types.StringType), withUrls, urlSet(t, "https://a.example")},
		"api change is read":        {urlSet(t, "https://b.example"), withUrls, urlSet(t, "https://a.example")},
		"out of band removal shown": {urlSet(t, "https://a.example"), withoutUrls, types.SetNull(types.StringType)},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := readToken(t, priorToken(t, tc.prior), tc.apiBody)
			if got == nil {
				t.Fatal("resource unexpectedly removed")
			}
			if !got.AllowedUrls.Equal(tc.want) {
				t.Errorf("allowed_urls = %s, want %s", got.AllowedUrls, tc.want)
			}
			// Secret token value is not returned by the list API and must be kept.
			if got.Token.ValueString() != "sk.stored" {
				t.Errorf("token = %s, want stored value", got.Token)
			}
		})
	}
}

func TestTokenRead_RemovedWhenMissing(t *testing.T) {
	if got := readToken(t, priorToken(t, types.SetNull(types.StringType)), `[{"id":"other","note":"x","scopes":[]}]`); got != nil {
		t.Errorf("expected resource to be removed from state, got %+v", got)
	}
}
