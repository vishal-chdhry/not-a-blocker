package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// TaggerDependsonRule checks whether tagger in image module has all tests in its dependson field
type OneTaggerPerRepoRule struct {
	tflint.DefaultRule
}

// NewTaggerDependsonRule returns a new rule
func NewOneTaggerPerRepoRule() *OneTaggerPerRepoRule {
	return &OneTaggerPerRepoRule{}
}

// Name returns the rule name
func (r *OneTaggerPerRepoRule) Name() string {
	return "one_tagger_per_repo_rule"
}

// Enabled returns whether the rule is enabled by default
func (r *OneTaggerPerRepoRule) Enabled() bool {
	return true
}

// Severity returns the rule severity
func (r *OneTaggerPerRepoRule) Severity() tflint.Severity {
	return tflint.ERROR
}

// Link returns the rule reference link
func (r *OneTaggerPerRepoRule) Link() string {
	return ""
}

// Check checks whether ...
func (r *OneTaggerPerRepoRule) Check(runner tflint.Runner) error {
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

	// find the tagger and publisher blocks
	taggerBlocks := make(map[string]*hclsyntax.Block)
	publisherBlocks := make(map[string]*hclsyntax.Block)
	for _, block := range blocks {
		if block.Type != "module" {
			continue
		}

		if srcExpr, found := block.Body.Attributes["source"]; !found {
			continue
		} else {
			var src string
			err := runner.EvaluateExpr(srcExpr.Expr, &src, nil)
			if err != nil {
				logger.Error("failed to evaluate tagger source value", "error", err)
				return err
			}
			if len(block.Labels) == 0 {
				return fmt.Errorf("invalid block with no labels %#v", block)
			}
			name := block.Labels[0]
			if strings.Contains(src, "/tflib/tagger") {
				if _, found := taggerBlocks[name]; found {
					return fmt.Errorf("tagger name conflict: %#v", taggerBlocks)
				}
				taggerBlocks[name] = block
			} else if strings.Contains(src, "/tflib/publisher") {
				if _, found := publisherBlocks[name]; found {
					return fmt.Errorf("tagger name conflict: %#v", publisherBlocks)
				}
				publisherBlocks[name] = block
			}

		}
	}

	logger.Debug("tagger blocks found", "data", fmt.Sprintf("%#v", taggerBlocks))
	logger.Debug("publisher blocks found", "data", fmt.Sprintf("%#v\n", publisherBlocks))

	if len(taggerBlocks) == 0 || len(publisherBlocks) == 0 {
		return nil
	}

	// multiple publishers can have the same target repo
	targetRepoToPublishers := make(map[string][]string)
	// a publisher can be in multiple taggers, if happens then throw error immediately
	publisherToTagger := make(map[string]string)
	// we have to ensure that one target repo is only pushed to by one tagger
	// use target repo -> publisher & publisher -> tagger to create a 1:1 map of target repo and tagger
	targetRepoToTagger := make(map[string]string)
	for name, publisher := range publisherBlocks {
		repo, ok := publisher.Body.Attributes["target_repository"]
		if !ok {
			err := runner.EmitIssue(
				r,
				fmt.Sprintf("publisher: %s does not have a target repository", name),
				publisher.Range(),
			)
			if err != nil {
				logger.Error("failed to emit issue", "error", err)
				return err
			}
		}

		publisherFileName := repo.Expr.Range().Filename
		targetRepositoryExpr := string(repo.Expr.Range().SliceBytes(files[publisherFileName].Bytes))
		if _, found := targetRepoToPublishers[targetRepositoryExpr]; found {
			targetRepoToPublishers[targetRepositoryExpr] = append(targetRepoToPublishers[targetRepositoryExpr], name)
		} else {
			targetRepoToPublishers[targetRepositoryExpr] = []string{name}
		}
	}

	for taggerName, tagger := range taggerBlocks {
		tags, ok := tagger.Body.Attributes["tags"]
		if !ok {
			err := runner.EmitIssue(
				r,
				fmt.Sprintf("tagger: %s does not have a tags attribute", taggerName),
				tagger.Range(),
			)
			if err != nil {
				logger.Error("failed to emit issue", "error", err)
				return err
			}
		}

		taggerFileName := tags.Expr.Range().Filename
		tagsExpr := string(tags.Expr.Range().SliceBytes(files[taggerFileName].Bytes))
		for publisher, pubBlock := range publisherBlocks {
			if regexp.MustCompile(fmt.Sprintf(`module\.%s[\[.)]`, publisher)).MatchString(tagsExpr) {
				if currentTagger, found := publisherToTagger[publisher]; !found {
					publisherToTagger[publisher] = taggerName
				} else {
					err := runner.EmitIssue(
						r,
						fmt.Sprintf("same publisher: %s cannot be in multiple taggers: %s, %s", publisher, currentTagger, taggerName),
						pubBlock.Range(),
					)
					if err != nil {
						logger.Error("failed to emit issue", "error", err)
						return err
					}
				}
			}
		}
	}

	fmt.Printf("%#v\n", targetRepoToPublishers)
	fmt.Printf("%#v\n", publisherToTagger)

	for targetRepo, publishers := range targetRepoToPublishers {
		for _, publisher := range publishers {
			tagger, found := publisherToTagger[publisher]
			if !found {
				continue // throwaway publisher
			}
			if currentTagger, found := targetRepoToTagger[targetRepo]; !found {
				targetRepoToTagger[targetRepo] = tagger
			} else if currentTagger != tagger {
				err := runner.EmitIssue(
					r,
					fmt.Sprintf("same target repository: %s cannot be in multiple taggers: %s, %s", targetRepo, currentTagger, tagger),
					taggerBlocks[tagger].Range(),
				)
				if err != nil {
					logger.Error("failed to emit issue", "error", err)
					return err
				}
			}
		}
	}

	logger.Debug("exiting rule", "name", r.Name())
	return nil
}
