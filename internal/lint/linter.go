package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-crucible/go-crucible/internal/types"
)

// lintRule is an internal check applied to a parsed YAML document.
type lintRule struct {
	name     string
	severity string
	check    func(file string, node *yaml.Node) []types.LintFinding
}

// builtinRules are the checks run against each workflow file.
var builtinRules = []lintRule{
	{
		name:     "workflow-name-required",
		severity: SeverityError,
		check: func(file string, node *yaml.Node) []types.LintFinding {
			if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
				return nil
			}
			root := node.Content[0]
			for i := 0; i+1 < len(root.Content); i += 2 {
				if root.Content[i].Value == "name" {
					val := root.Content[i+1].Value
					if strings.TrimSpace(val) != "" {
						return nil
					}
				}
			}
			return []types.LintFinding{
				newFinding(file, "workflow-name-required", SeverityError,
					"workflow is missing a non-empty 'name' field", 0),
			}
		},
	},
	{
		name:     "pin-actions-version",
		severity: SeverityWarning,
		check: func(file string, node *yaml.Node) []types.LintFinding {
			// Walk the document looking for 'uses:' nodes whose value ends in
			// a mutable tag like @v1, @latest or a branch name rather than a SHA.
			var findings []types.LintFinding
			walkNode(node, func(n *yaml.Node) {
				if n.Kind == yaml.MappingNode {
					for i := 0; i+1 < len(n.Content); i += 2 {
						key := n.Content[i]
						val := n.Content[i+1]
						if key.Value == "uses" && val.Kind == yaml.ScalarNode {
							ref := val.Value
							at := strings.LastIndex(ref, "@")
							if at < 0 {
								return
							}
							pin := ref[at+1:]
							// A SHA pin is 40 hex chars; anything shorter is mutable.
							if len(pin) < 40 {
								findings = append(findings, newFinding(
									file, "pin-actions-version", SeverityWarning,
									fmt.Sprintf("action %q is not pinned to a full SHA commit; use a 40-char SHA for reproducibility", ref),
									val.Line,
								))
							}
						}
					}
				}
			})
			return findings
		},
	},
	{
		name:     "no-plaintext-secret",
		severity: SeverityError,
		check: func(file string, node *yaml.Node) []types.LintFinding {
			var findings []types.LintFinding
			suspectKeys := []string{"password", "token", "secret", "api_key", "apikey"}
			walkNode(node, func(n *yaml.Node) {
				if n.Kind == yaml.MappingNode {
					for i := 0; i+1 < len(n.Content); i += 2 {
						key := n.Content[i].Value
						val := n.Content[i+1]
						for _, sk := range suspectKeys {
							if strings.EqualFold(key, sk) && val.Kind == yaml.ScalarNode {
								v := val.Value
								// Flag if it looks like a literal secret rather than a ${{ secrets.* }} reference.
								if !strings.HasPrefix(v, "${{") && v != "" {
									findings = append(findings, newFinding(
										file, "no-plaintext-secret", SeverityError,
										fmt.Sprintf("key %q appears to contain a plaintext secret value", key),
										val.Line,
									))
								}
							}
						}
					}
				}
			})
			return findings
		},
	},
}

// walkNode calls fn for n and every descendant node.
func walkNode(n *yaml.Node, fn func(*yaml.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, child := range n.Content {
		walkNode(child, fn)
	}
}

// LintWorkflows walks dir, lints every *.yml and *.yaml file it finds, and
// returns the aggregated findings.
func LintWorkflows(dir string) ([]types.LintFinding, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lint: reading directory %q: %w", dir, err)
	}

	var findings []types.LintFinding

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}

		path := filepath.Join(dir, name)

		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("lint: opening %q: %w", path, err)
		}
		defer f.Close()

		var doc yaml.Node
		dec := yaml.NewDecoder(f)
		if err := dec.Decode(&doc); err != nil {
			// Skip files that aren't valid YAML.
			continue
		}

		for _, rule := range builtinRules {
			ruleFindings := rule.check(path, &doc)
			findings = append(findings, ruleFindings...)
		}
	}

	return findings, nil
}
