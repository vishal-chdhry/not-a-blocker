package rules_test

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
	"github.com/vishal-chdhry/not-a-blocker/rules"
)

var one_tagger_tfmodpass = `
module "agent-versioned" {
  source = "../../tflib/publisher"

  name              = basename(path.module)
  target_repository = var.target_repository
  config            = module.agent-config.config
  build-dev         = true
  main_package      = "datadog-agent"
}


module "agent-jmx-versioned" {
  source = "../../tflib/publisher"

  name              = basename(path.module)
  target_repository = var.target_repository
  config            = module.agent-jmx-config.config
  build-dev         = true
  main_package      = "datadog-agent-jmx"
  tags_suffix       = "-jmx"
}

module "agent-tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test]
  tags       = module.agent-versioned.latest_tag_map
}
`

var one_tagger_tfmodfail = `
module "agent-versioned" {
  source = "../../tflib/publisher"

  name              = basename(path.module)
  target_repository = var.target_repository
  config            = module.agent-config.config
  build-dev         = true
  main_package      = "datadog-agent"
}


module "agent-jmx-versioned" {
  source = "../../tflib/publisher"

  name              = basename(path.module)
  target_repository = var.target_repository
  config            = module.agent-jmx-config.config
  build-dev         = true
  main_package      = "datadog-agent-jmx"
  tags_suffix       = "-jmx"
}

module "agent-tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test]
  tags       = module.agent-versioned.latest_tag_map
}

module "agent-jmx-tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test-jmx]
  tags = {
    for k, v in module.agent-jmx-versioned.latest_tag_map :
    (startswith(k, "latest") ? replace(k, "latest", "latest-jmx") : k) => v
  }
}
`

func Test_TerraformOneTaggerPerRepo(t *testing.T) {
	tests := []struct {
		Name     string
		Content  string
		Expected helper.Issues
	}{
		{
			Name:     "pass test",
			Content:  tagger_dependson_tfmodpass,
			Expected: helper.Issues{},
		},
		{
			Name:    "fail test",
			Content: tagger_dependson_tfmodfail,
			Expected: helper.Issues{
				{
					Rule:    rules.NewTaggerDependsonRule(),
					Message: "test: test-dev-versioned is not mentioned in tagger's depends_on",
					Range: hcl.Range{
						Filename: "main.tf",
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
			runner := helper.TestRunner(t, map[string]string{"main.tf": test.Content})

			if err := rule.Check(runner); err != nil {
				t.Fatalf("Unexpected error occurred: %s", err)
			}

			helper.AssertIssues(t, test.Expected, runner.Issues)
		})
	}
}
