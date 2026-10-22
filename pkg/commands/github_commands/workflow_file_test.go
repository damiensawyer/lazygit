package github_commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWorkflowDispatchInputs(t *testing.T) {
	scenarios := []struct {
		name          string
		content       string
		expected      []WorkflowInput
		expectedError error
	}{
		{
			name:     "single trigger",
			content:  "on: workflow_dispatch\n",
			expected: nil,
		},
		{
			name:     "list of triggers",
			content:  "on: [push, workflow_dispatch]\n",
			expected: nil,
		},
		{
			name:          "not dispatchable",
			content:       "on:\n  push:\n    branches: [main]\n",
			expectedError: ErrNotDispatchable,
		},
		{
			name: "inputs",
			content: `on:
  push:
  workflow_dispatch:
    inputs:
      environment:
        description: Where to deploy
        required: true
        type: choice
        options: [staging, production]
      dry_run:
        type: boolean
        default: true
      version:
        description: Version to release
`,
			expected: []WorkflowInput{
				{Name: "environment", Description: "Where to deploy", Required: true, Type: "choice", Options: []string{"staging", "production"}},
				{Name: "dry_run", Type: "boolean", Default: "true"},
				{Name: "version", Description: "Version to release", Type: "string"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			inputs, err := ParseWorkflowDispatchInputs(s.content)
			assert.Equal(t, s.expectedError, err)
			assert.Equal(t, s.expected, inputs)
		})
	}
}
