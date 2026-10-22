package github_commands

import (
	"errors"
	"slices"

	"gopkg.in/yaml.v3"
)

// WorkflowInput is an input a workflow takes when it is run manually.
type WorkflowInput struct {
	Name        string
	Description string
	Required    bool
	Default     string
	// "string", "boolean", "choice", "number" or "environment"
	Type    string
	Options []string
}

var ErrNotDispatchable = errors.New("workflow has no workflow_dispatch trigger")

// ParseWorkflowDispatchInputs reads a workflow definition and returns the
// inputs of its workflow_dispatch trigger, in the order they are defined in,
// or ErrNotDispatchable if it can't be run manually.
func ParseWorkflowDispatchInputs(content string) ([]WorkflowInput, error) {
	var workflow struct {
		On yaml.Node `yaml:"on"`
	}
	if err := yaml.Unmarshal([]byte(content), &workflow); err != nil {
		return nil, err
	}

	// The triggers can be given as a single event name, a list of them, or a
	// mapping from event name to its configuration.
	on := &workflow.On
	switch on.Kind {
	case yaml.ScalarNode:
		if on.Value == "workflow_dispatch" {
			return nil, nil
		}
	case yaml.SequenceNode:
		if slices.ContainsFunc(on.Content, func(node *yaml.Node) bool { return node.Value == "workflow_dispatch" }) {
			return nil, nil
		}
	case yaml.MappingNode:
		if dispatch := mappingValue(on, "workflow_dispatch"); dispatch != nil {
			return parseWorkflowInputs(dispatch)
		}
	case yaml.DocumentNode, yaml.AliasNode:
	}
	return nil, ErrNotDispatchable
}

func parseWorkflowInputs(dispatch *yaml.Node) ([]WorkflowInput, error) {
	inputsNode := mappingValue(dispatch, "inputs")
	if inputsNode == nil || inputsNode.Kind != yaml.MappingNode {
		return nil, nil
	}

	inputs := make([]WorkflowInput, 0, len(inputsNode.Content)/2)
	for i := 0; i+1 < len(inputsNode.Content); i += 2 {
		var spec struct {
			Description string   `yaml:"description"`
			Required    bool     `yaml:"required"`
			Default     string   `yaml:"default"`
			Type        string   `yaml:"type"`
			Options     []string `yaml:"options"`
		}
		if err := inputsNode.Content[i+1].Decode(&spec); err != nil {
			return nil, err
		}
		if spec.Type == "" {
			spec.Type = "string"
		}
		inputs = append(inputs, WorkflowInput{
			Name:        inputsNode.Content[i].Value,
			Description: spec.Description,
			Required:    spec.Required,
			Default:     spec.Default,
			Type:        spec.Type,
			Options:     spec.Options,
		})
	}
	return inputs, nil
}
