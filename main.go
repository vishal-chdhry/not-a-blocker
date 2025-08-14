package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/vishal-chdhry/not-a-blocker/rules"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "not-a-blocker",
			Version: "latest",
			Rules: []tflint.Rule{
				rules.NewTaggerDependsonRule(),
				rules.NewOneTaggerPerRepoRule(),
			},
		},
	})
}
