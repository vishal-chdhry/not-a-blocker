package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/vishal-chdhry/not-a-blocker/rules"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "tflint-ruleset-not-a-blocker",
			Version: "latest",
			Rules: []tflint.Rule{
				rules.NewTaggerDependsonRule(),
			},
		},
	})
}
