// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gopkg.in/h2non/gock.v1"
)

func init() {
	if os.Getenv("MAPBOX_USERNAME") == "" {
		if err := os.Setenv("MAPBOX_USERNAME", "test-token"); err != nil {
			panic(fmt.Sprintf("set MAPBOX_USERNAME: %v", err))
		}

		if err := os.Setenv("MOCK", "1"); err != nil {
			panic(fmt.Sprintf("set MOCK: %v", err))
		}
	}
}

func TestAccTokenResource_basic(t *testing.T) {
	resourceName := "mapbox_token.test"
	username := os.Getenv("MAPBOX_USERNAME")
	note := "test-note"

	if os.Getenv("MOCK") != "" {
		tokenEndpoint := fmt.Sprintf("tokens/v2/%s", username)
		var id = "cmihkow060gbm3fs8s44zh5v7"
		var token = "pk.eyJ1Ijoi9WRtaW4tY3OuYW5hbGFiIiwiYSh6ImNtaWl3bDhraTBjYmozbXI0sj03ZnFuNDkikW.qLjLLJ4TTQ5VYrHHgwyY3g"
		defer gock.OffAll()

		gock.New("https://api.mapbox.com").
			Post(tokenEndpoint).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusOK).
			JSON(tokenCreateBody{
				Note:        note,
				AllowedUrls: []string{"https://docs.mapbox.com"},
				Id:          &id,
				Scopes:      []string{"styles:read", "fonts:read"},
				Token:       &token,
			})

		gock.New("https://api.mapbox.com").
			Get(tokenEndpoint).
			MatchParam("access_token", "test-token").
			Times(3).
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

		gock.New("https://api.mapbox.com").
			Patch(fmt.Sprintf("%s/%s", tokenEndpoint, id)).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusOK).
			JSON(tokenCreateBody{
				Note:        note,
				AllowedUrls: []string{"https://docs.mapbox1.com"},
				Id:          &id,
				Scopes:      []string{"fonts:read"},
				Token:       &token,
			})

		gock.New("https://api.mapbox.com").
			Get(tokenEndpoint).
			MatchParam("access_token", "test-token").
			Times(1).
			Reply(http.StatusOK).
			JSON([]tokenCreateBody{
				{
					Note:        note,
					AllowedUrls: []string{"https://docs.mapbox1.com"},
					Id:          &id,
					Scopes:      []string{"styles:read", "fonts:read"},
					Token:       &token,
				},
			})

		gock.New("https://api.mapbox.com").
			Delete(fmt.Sprintf("%s/%s", tokenEndpoint, id)).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusNoContent)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTokenResourceConfig(username, note, "https://docs.mapbox.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "note", note),
					resource.TestCheckResourceAttr(resourceName, "scopes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "fonts:read"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "styles:read"),
					resource.TestCheckResourceAttr(resourceName, "allowed_urls.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "allowed_urls.*", "https://docs.mapbox.com"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// // Update and Read testing
			{
				Config: testAccTokenResourceConfig(username, note, "https://docs.mapbox1.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "note", note),
					resource.TestCheckResourceAttr(resourceName, "scopes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "fonts:read"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "styles:read"),
					resource.TestCheckResourceAttr(resourceName, "allowed_urls.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "allowed_urls.*", "https://docs.mapbox1.com"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccTokenResource_empty(t *testing.T) {
	resourceName := "mapbox_token.test"
	username := os.Getenv("MAPBOX_USERNAME")
	note := "test-note"

	if os.Getenv("MOCK") != "" {
		tokenEndpoint := fmt.Sprintf("tokens/v2/%s", username)
		var id = "cmihkow060gbm3fs8s44zh5v7"
		var token = "pk.eyJ1Ijoi9WRtaW4tY3OuYW5hbGFiIiwiYSh6ImNtaWl3bDhraTBjYmozbXI0sj03ZnFuNDkikW.qLjLLJ4TTQ5VYrHHgwyY3g"
		defer gock.OffAll()

		gock.New("https://api.mapbox.com").
			Post(tokenEndpoint).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusOK).
			JSON(tokenCreateBody{
				Note:        note,
				AllowedUrls: []string{},
				Id:          &id,
				Scopes:      []string{"styles:read", "fonts:read"},
				Token:       &token,
			})

		gock.New("https://api.mapbox.com").
			Get(tokenEndpoint).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusOK).
			JSON([]tokenCreateBody{
				{
					Note:        note,
					AllowedUrls: []string{},
					Id:          &id,
					Scopes:      []string{"styles:read", "fonts:read"},
					Token:       &token,
				},
			})

		gock.New("https://api.mapbox.com").
			Delete(fmt.Sprintf("%s/%s", tokenEndpoint, id)).
			MatchParam("access_token", "test-token").
			Persist().
			Reply(http.StatusNoContent)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTokenResourceConfigEmpty(username, note),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", username),
					resource.TestCheckResourceAttr(resourceName, "note", note),
					resource.TestCheckResourceAttr(resourceName, "scopes.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "fonts:read"),
					resource.TestCheckTypeSetElemAttr(resourceName, "scopes.*", "styles:read"),
					resource.TestCheckResourceAttr(resourceName, "allowed_urls.#", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTokenResourceConfig(username, note, url string) string {
	return fmt.Sprintf(`
resource "mapbox_token" "test" {
  username     = %[1]q
  note         = %[2]q
  scopes       = ["styles:read", "fonts:read"]
  allowed_urls = ["%[3]s"]
}
`, username, note, url)
}

func testAccTokenResourceConfigEmpty(username, note string) string {
	return fmt.Sprintf(`
resource "mapbox_token" "test" {
  username     = %[1]q
  note         = %[2]q
  scopes       = ["styles:read", "fonts:read"]
}
`, username, note)
}
