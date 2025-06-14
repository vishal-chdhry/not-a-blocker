package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
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
	// collect all resources under module.tagger
	modules, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type:       "module",
				LabelNames: []string{"*"},
				Body: &hclext.BodySchema{
					Mode: hclext.SchemaJustAttributesMode,
				},
			},
		},
	}, nil)
	if err != nil {
		logger.Error("failed to get module contents", "error", err)
		return err
	}

	logger.Debug("blocks found", "count", len(modules.Blocks))

	// find the tests and tagger blocks
	var taggerBlock *hclext.Block
	testBlocks := make(map[string]*hclext.Block)
	for _, block := range modules.Blocks {
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
			if strings.HasPrefix(label, "test-") {
				testBlocks[label] = block
			}
		}
	}

	logger.Debug("tagger found", "data", fmt.Sprintf("%#v", taggerBlock))
	logger.Debug("test blocks found", "data", fmt.Sprintf("%#v\n", testBlocks))

	var checkedTests []string
	if taggerBlock != nil {
		traversals := taggerBlock.Body.Attributes["depends_on"].Expr.Variables()
		for _, t := range traversals {
			checkedTests = append(checkedTests, t[1].(hcl.TraverseAttr).Name)
		}
	}

	logger.Debug("checked tests", "data", fmt.Sprintf("%#v\n", checkedTests))

	for name, test := range testBlocks {
		if !slices.Contains(checkedTests, name) {
			err := runner.EmitIssue(
				r,
				fmt.Sprintf("test: %s is not mentioned in tagger's depends_on", name),
				test.DefRange,
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
