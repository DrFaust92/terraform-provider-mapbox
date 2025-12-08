// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func init() {
	if os.Getenv("MAPBOX_ACCESS_TOKEN") == "" {
		if err := os.Setenv("MAPBOX_ACCESS_TOKEN", "test-token"); err != nil {
			panic(fmt.Sprintf("set MAPBOX_ACCESS_TOKEN: %v", err))
		}

		if err := os.Setenv("MOCK", "1"); err != nil {
			panic(fmt.Sprintf("set MOCK: %v", err))
		}
	}
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mapbox": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}
