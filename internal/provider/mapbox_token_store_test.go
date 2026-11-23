// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestTokenRead_StoreTokenFalseKeepsValueOutOfState(t *testing.T) {
	prior := priorToken(t, types.SetNull(types.StringType))
	prior.StoreToken = types.BoolValue(false)
	prior.Token = types.StringNull()

	got := readToken(t, prior, `[{"id":"abc","note":"note","scopes":["styles:read"],"token":"pk.value","usage":"pk"}]`)
	if got == nil {
		t.Fatal("resource unexpectedly removed")
	}
	if !got.Token.IsNull() {
		t.Errorf("token = %s, want null", got.Token)
	}
}

func TestTokenRead_ImportDefaultsStoreToken(t *testing.T) {
	prior := priorToken(t, types.SetNull(types.StringType))
	prior.Token = types.StringNull()

	got := readToken(t, prior, `[{"id":"abc","note":"note","scopes":["styles:read"],"token":"pk.value","usage":"pk"}]`)
	if got == nil {
		t.Fatal("resource unexpectedly removed")
	}
	if !got.StoreToken.ValueBool() {
		t.Errorf("store_token = %s, want true", got.StoreToken)
	}
	if got.Token.ValueString() != "pk.value" {
		t.Errorf("token = %s, want pk.value", got.Token)
	}
}

func TestTokenValidateConfig_StoreToken(t *testing.T) {
	ctx := context.Background()
	r := &TokenResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	scopes := func(elems ...attr.Value) types.Set {
		return types.SetValueMust(types.StringType, elems)
	}

	cases := map[string]struct {
		storeToken types.Bool
		scopes     types.Set
		wantErr    bool
	}{
		"public scopes allowed":       {types.BoolValue(false), scopes(types.StringValue("styles:read"), types.StringValue("fonts:read")), false},
		"secret scope rejected":       {types.BoolValue(false), scopes(types.StringValue("styles:read"), types.StringValue("tokens:write")), true},
		"secret scope stored is fine": {types.BoolValue(true), scopes(types.StringValue("tokens:write")), false},
		"unset defaults to storing":   {types.BoolNull(), scopes(types.StringValue("tokens:write")), false},
		"unknown scope skipped":       {types.BoolValue(false), scopes(types.StringUnknown()), false},
		"unknown scopes skipped":      {types.BoolValue(false), types.SetUnknown(types.StringType), false},
		"unknown store_token skipped": {types.BoolUnknown(), scopes(types.StringValue("tokens:write")), false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, TokenResourceModel{
				AllowedUrls: types.SetNull(types.StringType),
				Id:          types.StringNull(),
				Note:        types.StringValue("note"),
				Scopes:      tc.scopes,
				StoreToken:  tc.storeToken,
				Token:       types.StringNull(),
				Username:    types.StringValue("user"),
			}); diags.HasError() {
				t.Fatalf("build config: %v", diags)
			}

			var resp resource.ValidateConfigResponse
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw}}, &resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("error = %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func openEphemeralToken(t *testing.T, apiBody string) (TokenEphemeralResourceModel, ephemeral.OpenResponse) {
	t.Helper()
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(apiBody))
	}))
	defer server.Close()

	accessToken := "sk.test"
	r := &TokenEphemeralResource{client: &Client{AccessToken: &accessToken, BaseURL: server.URL + "/"}}

	var schemaResp ephemeral.SchemaResponse
	r.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)
	tfType := schemaResp.Schema.Type().TerraformType(ctx)

	config := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tfType, nil)}
	if diags := config.Set(ctx, TokenEphemeralResourceModel{
		Username: types.StringValue("user"),
		TokenId:  types.StringValue("abc"),
		Token:    types.StringNull(),
	}); diags.HasError() {
		t.Fatalf("build config: %v", diags)
	}

	resp := ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData{Schema: schemaResp.Schema, Raw: tftypes.NewValue(tfType, nil)}}
	r.Open(ctx, ephemeral.OpenRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: config.Raw}}, &resp)

	var got TokenEphemeralResourceModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.Result.Get(ctx, &got); diags.HasError() {
			t.Fatalf("get result: %v", diags)
		}
	}
	return got, resp
}

func TestTokenEphemeralOpen(t *testing.T) {
	t.Run("public token value returned", func(t *testing.T) {
		got, resp := openEphemeralToken(t, `[{"id":"abc","note":"n","scopes":["styles:read"],"token":"pk.value"}]`)
		if resp.Diagnostics.HasError() {
			t.Fatalf("open: %v", resp.Diagnostics)
		}
		if got.Token.ValueString() != "pk.value" {
			t.Errorf("token = %s, want pk.value", got.Token)
		}
	})

	t.Run("secret token errors", func(t *testing.T) {
		_, resp := openEphemeralToken(t, `[{"id":"abc","note":"n","scopes":["tokens:write"],"usage":"sk"}]`)
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "Secret tokens") {
			t.Errorf("expected secret token error, got %v", resp.Diagnostics)
		}
	})

	t.Run("missing token errors", func(t *testing.T) {
		_, resp := openEphemeralToken(t, `[{"id":"other","note":"n","scopes":[],"token":"pk.other"}]`)
		if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != "Token Not Found" {
			t.Errorf("expected not found error, got %v", resp.Diagnostics)
		}
	})
}
