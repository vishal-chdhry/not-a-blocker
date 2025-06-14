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
  depends_on = [module.test-versioned]
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
