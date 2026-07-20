// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gopkg.in/h2non/gock.v1"
)

func TestAccTokenListResource_basic(t *testing.T) {
	username := os.Getenv("MAPBOX_USERNAME")
	note := "test-note"

	if os.Getenv("MOCK") != "" {
		tokenEndpoint := fmt.Sprintf("tokens/v2/%s", username)
		var id = "cmihkow060gbm3fs8s44zh5v7"
		var token = "pk.eyJ1Ijoi9WRtaW4tY3OuYW5hbGFiIiwiYSh6ImNtaWl3bDhraTBjYmozbXI0sj03ZnFuNDkikW.qLjLLJ4TTQ5VYrHHgwyY3g"
		defer gock.OffAll()

		gock.New("https://api.mapbox.com").
			Get(tokenEndpoint).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusOK).
			JSON([]tokenCreateBody{
				{
					Note:        note,
					AllowedUrls: []string{"https://docs.mapbox.com"},
					Id:          &id,
					Scopes:      []string{"styles:read", "fonts:read"},
					Token:       &token,
				},
			})
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Query: true,
				Config: fmt.Sprintf(`
provider "mapbox" {}

list "mapbox_token" "test" {
  provider         = mapbox
  include_resource = true

  config {
    username = %[1]q
  }
}
`, username),
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLength("mapbox_token.test", 1),
					querycheck.ExpectIdentity("mapbox_token.test", map[string]knownvalue.Check{
						"token_id": knownvalue.StringExact("cmihkow060gbm3fs8s44zh5v7"),
						"username": knownvalue.StringExact(username),
					}),
					querycheck.ExpectResourceKnownValues(
						"mapbox_token.test",
						queryfilter.ByResourceIdentity(map[string]knownvalue.Check{
							"token_id": knownvalue.StringExact("cmihkow060gbm3fs8s44zh5v7"),
							"username": knownvalue.StringExact(username),
						}),
						[]querycheck.KnownValueCheck{
							{
								Path:       tfjsonpath.New("note"),
								KnownValue: knownvalue.StringExact(note),
							},
							{
								Path: tfjsonpath.New("scopes"),
								KnownValue: knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("styles:read"),
									knownvalue.StringExact("fonts:read"),
								}),
							},
						},
					),
				},
			},
		},
	})
}
