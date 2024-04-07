// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTokenResource_basic(t *testing.T) {
	resourceName := "mapbox_token.test"
	name := os.Getenv("MAPBOX_USERNAME")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTokenResourceConfig(name, "https://docs.mapbox.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", name),
					resource.TestCheckResourceAttr(resourceName, "note", name),
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
			// Update and Read testing
			{
				Config: testAccTokenResourceConfig(name, "https://docs.mapbox1.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", name),
					resource.TestCheckResourceAttr(resourceName, "note", name),
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
	name := os.Getenv("MAPBOX_USERNAME")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccTokenResourceConfigEmpty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "username", name),
					resource.TestCheckResourceAttr(resourceName, "note", name),
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

func testAccTokenResourceConfig(name, url string) string {
	return fmt.Sprintf(`
resource "mapbox_token" "test" {
  username     = %[1]q
  note         = %[1]q
  scopes       = ["styles:read", "fonts:read"]
  allowed_urls = ["%[2]s"]
}
`, name, url)
}

func testAccTokenResourceConfigEmpty(name string) string {
	return fmt.Sprintf(`
resource "mapbox_token" "test" {
  username     = %[1]q
  note         = %[1]q
  scopes       = ["styles:read", "fonts:read"]
}
`, name)
}
