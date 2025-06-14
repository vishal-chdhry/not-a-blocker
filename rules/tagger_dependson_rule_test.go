package rules_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
	"github.com/vishal-chdhry/not-a-blocker/rules"
)

var tfmodpass = `
module "test-versioned" {
  for_each          = local.elixir_versions
  source            = "./tests"
  digest            = module.versioned[each.key].image_ref
  target_repository = var.target_repository
  image_version     = each.key
}

module "tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test-versioned]
  tags       = merge([for v in module.versioned : v.latest_tag_map]...)
}
`

var tfmodfail = `
terraform {
  required_providers {
    oci = { source = "chainguard-dev/oci" }
  }
}

variable "target_repository" {
  description = "The docker repo into which the image and attestations should be published."
}

module "versions" {
  source  = "../../tflib/versions"
  package = "elixir"
}

locals {
  # List of versions to skip
  # 1.14 is no longer supported
  # FIXME: remove this once this isnt on the package metadata anymore
  # TODO: Fix 1.15-1.17
  ignore_versions = ["1.14", "1.15", "1.16", "1.17"]

  # Create list of versions excluding ignored ones
  elixir_versions = {
    for k, v in module.versions.versions :
    k => v if !contains(local.ignore_versions, v.version)
  }
}

module "config" {
  for_each = local.elixir_versions
  source   = "./config"
  extra_packages = [
    each.key,
    "busybox", # Elixir depends on some coreutils utility, when this is not here it fails with exec (elixir path) not found
    "rebar3"   # Need to include most things from the upstream image + environment variable (REBAR_VERSION) is set to a valid version with this
  ]
}

module "versioned" {
  for_each          = local.elixir_versions
  source            = "../../tflib/publisher"
  eol               = each.value.eol
  name              = basename(path.module)
  target_repository = var.target_repository
  config            = module.config[each.key].config
  build-dev         = true
  main_package      = each.value.main
  update-repo       = each.value.is_latest
  # Most external packages need -dev erlang/elixir headers to actually be useful
  # We fetch the package from the dummy config on config/* so that we can avoid skews if that happens at some point
  extra_dev_packages = ["erlang-${module.config[each.key].versions.erlang_major_version}-dev"]
}

module "test-versioned" {
  for_each          = local.elixir_versions
  source            = "./tests"
  digest            = module.versioned[each.key].image_ref
  target_repository = var.target_repository
  image_version     = each.key
}

module "test-dev-versioned" {
  for_each          = local.elixir_versions
  source            = "./tests"
  check-dev         = true
  digest            = module.versioned[each.key].dev_ref
  target_repository = var.target_repository
  image_version     = "${each.key}-dev"
}

module "tagger" {
  source     = "../../tflib/tagger"
  depends_on = [lo.tdfest-dev-versioned]
  tags       = merge([for v in module.versioned : v.latest_tag_map]...)
}
`

func Test_TerraformBackendType(t *testing.T) {
	tests := []struct {
		Name     string
		Content  string
		Expected helper.Issues
	}{
		{
			Name:     "pass test",
			Content:  tfmodpass,
			Expected: helper.Issues{},
		},
		{
			Name:    "fail test",
			Content: tfmodfail,
			Expected: helper.Issues{
				{
					Rule:    rules.NewTaggerDependsonRule(),
					Message: "test: test-dev-versioned is not mentioned in tagger's depends_on",
					Range: hcl.Range{
						Filename: "resource.tf",
						Start:    hcl.Pos{Line: 10, Column: 1},
						End:      hcl.Pos{Line: 10, Column: 28},
					},
				},
			},
		},
	}

	rule := rules.NewTaggerDependsonRule()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			runner := helper.TestRunner(t, map[string]string{"resource.tf": test.Content})

			if err := rule.Check(runner); err != nil {
				t.Fatalf("Unexpected error occurred: %s", err)
			}

			helper.AssertIssues(t, test.Expected, runner.Issues)
		})
	}
}
