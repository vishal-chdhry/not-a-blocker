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

module "tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test]
  tags = merge(
    module.agent-versioned.latest_tag_map,
    {
      for k, v in module.agent-jmx-versioned.latest_tag_map :
      (startswith(k, "latest") ? replace(k, "latest", "latest-jmx") : k) => v
    }
  )
}

module "cluster-agent-versioned" {
  source = "../../tflib/publisher"

  name = basename(path.module)
  # ensure this is "/datadog-cluster-agent" instead of "/datadog-agent-cluster"
  target_repository = "${split("-agent", var.target_repository)[0]}-cluster-agent"
  config            = module.cluster-agent-config.config
  build-dev         = true
  main_package      = "datadog-cluster-agent"
}

module "cluster-agent-tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test]
  tags       = module.cluster-agent-versioned.latest_tag_map
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

module "cluster-agent-versioned" {
  source = "../../tflib/publisher"

  name = basename(path.module)
  # ensure this is "/datadog-cluster-agent" instead of "/datadog-agent-cluster"
  target_repository = "${split("-agent", var.target_repository)[0]}-cluster-agent"
  config            = module.cluster-agent-config.config
  build-dev         = true
  main_package      = "datadog-cluster-agent"
}

module "cluster-agent-tagger" {
  source     = "../../tflib/tagger"
  depends_on = [module.test]
  tags       = module.cluster-agent-versioned.latest_tag_map
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
			Content:  one_tagger_tfmodpass,
			Expected: helper.Issues{},
		},
		{
			Name:    "fail test",
			Content: one_tagger_tfmodfail,
			Expected: helper.Issues{
				{
					Rule:    rules.NewOneTaggerPerRepoRule(),
					Message: "same target repository: var.target_repository cannot be in multiple taggers: agent-tagger, agent-jmx-tagger",
					Range: hcl.Range{
						Filename: "main.tf",
						Start:    hcl.Pos{Line: 47, Column: 1},
						End:      hcl.Pos{Line: 54, Column: 2},
					},
				},
			},
		},
	}

	rule := rules.NewOneTaggerPerRepoRule()

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
