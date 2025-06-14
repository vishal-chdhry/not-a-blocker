package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// TaggerDependsonRule checks whether tagger in image module has all tests in its dependson field
type TaggerDependsonRule struct {
	tflint.DefaultRule
}

// NewTaggerDependsonRule returns a new rule
func NewTaggerDependsonRule() *TaggerDependsonRule {
	return &TaggerDependsonRule{}
}

// Name returns the rule name
func (r *TaggerDependsonRule) Name() string {
	return "tagger_dependson_rule"
}

// Enabled returns whether the rule is enabled by default
func (r *TaggerDependsonRule) Enabled() bool {
	return true
}

// Severity returns the rule severity
func (r *TaggerDependsonRule) Severity() tflint.Severity {
	return tflint.ERROR
}

// Link returns the rule reference link
func (r *TaggerDependsonRule) Link() string {
	return ""
}

// Check checks whether ...
func (r *TaggerDependsonRule) Check(runner tflint.Runner) error {
	logger.Debug("checking rule", "name", r.Name())
	var blocks hclsyntax.Blocks

	files, err := runner.GetFiles()
	if err != nil {
		logger.Error("failed to fetch files", "error", err)
		return err
	}

	for _, file := range files {
		b := file.Body.(*hclsyntax.Body).Blocks
		blocks = append(blocks, b...)
	}

	logger.Debug("blocks found", "count", len(blocks))

	// find the tests and tagger blocks
	var taggerBlock *hclsyntax.Block
	testBlocks := make(map[string]*hclsyntax.Block)
	for _, block := range blocks {
		if block.Type != "module" {
			continue
		}

		if slices.Contains(block.Labels, "tagger") {
			var src string
			err := runner.EvaluateExpr(block.Body.Attributes["source"].Expr, &src, nil)
			if err != nil {
				logger.Error("failed to evaluate tagger source value", "error", err)
				return err
			}
			if strings.Contains(src, "/tflib/tagger") {
				taggerBlock = block
			}
		}

		for _, label := range block.Labels {
			if strings.HasPrefix(label, "test") {
				testBlocks[label] = block
			}
		}
	}

	logger.Debug("tagger found", "data", fmt.Sprintf("%#v", taggerBlock))
	logger.Debug("test blocks found", "data", fmt.Sprintf("%#v\n", testBlocks))

	if taggerBlock == nil {
		return nil
	}

	var checkedTests []string
	dependsOn, ok := taggerBlock.Body.Attributes["depends_on"]
	if !ok {
		if len(testBlocks) > 0 {
			for name, test := range testBlocks {
				err := runner.EmitIssue(
					r,
					fmt.Sprintf("test: %s is present but tagger has no depends_on attribute", name),
					test.DefRange(),
				)
				if err != nil {
					logger.Error("failed to emit issue", "error", err)
					return err
				}
			}
		}
		return nil // dont worry about the case with no depends_on and tests
	}

	traversals := dependsOn.Expr.Variables()
	for _, t := range traversals {
		checkedTests = append(checkedTests, t[1].(hcl.TraverseAttr).Name)
	}

	logger.Debug("checked tests", "data", fmt.Sprintf("%#v\n", checkedTests))

	for name, test := range testBlocks {
		if !slices.Contains(checkedTests, name) {
			err := runner.EmitIssue(
				r,
				fmt.Sprintf("test: %s is not mentioned in tagger's depends_on", name),
				test.DefRange(),
			)
			if err != nil {
				logger.Error("failed to emit issue", "error", err)
				return err
			}
		}
	}

	logger.Debug("exiting rule", "name", r.Name())
	return nil
}
