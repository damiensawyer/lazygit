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

func TestParseWorkflowDispatchInputsTriggers(t *testing.T) {
	scenarios := []struct {
		name          string
		content       string
		expected      []WorkflowInput
		expectedError error
	}{
		{
			name:          "empty file",
			content:       "",
			expectedError: ErrNotDispatchable,
		},
		{
			name:          "no triggers",
			content:       "name: CI\njobs:\n  build:\n    runs-on: ubuntu-latest\n",
			expectedError: ErrNotDispatchable,
		},
		{
			name:          "single other trigger",
			content:       "on: push\n",
			expectedError: ErrNotDispatchable,
		},
		{
			name:          "list of other triggers",
			content:       "on:\n  - push\n  - pull_request\n",
			expectedError: ErrNotDispatchable,
		},
		{
			name:     "trigger without configuration",
			content:  "on:\n  push:\n  workflow_dispatch:\n",
			expected: nil,
		},
		{
			name:     "trigger with empty configuration",
			content:  "on:\n  workflow_dispatch: {}\n",
			expected: nil,
		},
		{
			name:     "inputs that aren't a mapping",
			content:  "on:\n  workflow_dispatch:\n    inputs: [a, b]\n",
			expected: nil,
		},
		{
			name:     "no inputs",
			content:  "on:\n  workflow_dispatch:\n    inputs: {}\n",
			expected: []WorkflowInput{},
		},
		{
			name:          "a similarly named trigger",
			content:       "on:\n  workflow_call:\n    inputs:\n      x:\n        type: string\n",
			expectedError: ErrNotDispatchable,
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

func TestParseWorkflowDispatchInputsOfEveryType(t *testing.T) {
	content := `name: Deploy
on:
  workflow_dispatch:
    inputs:
      name:
        description: Who to greet
        default: world
      verbose:
        type: boolean
        default: false
      retries:
        type: number
        default: 3
      ratio:
        type: number
        default: 0.5
      target:
        type: environment
        required: true
      level:
        type: choice
        description: Log level
        options:
          - info
          - warning
          - debug
        default: warning
      quoted:
        type: string
        default: "  spaced  "
      multiline:
        description: |
          First line
          second line
      bare:
      numeric_options:
        type: choice
        options: [1, 2, 3]
jobs: {}
`

	inputs, err := ParseWorkflowDispatchInputs(content)
	assert.NoError(t, err)

	assert.Equal(t, []WorkflowInput{
		{Name: "name", Description: "Who to greet", Default: "world", Type: "string"},
		{Name: "verbose", Default: "false", Type: "boolean"},
		{Name: "retries", Default: "3", Type: "number"},
		{Name: "ratio", Default: "0.5", Type: "number"},
		{Name: "target", Required: true, Type: "environment"},
		{Name: "level", Description: "Log level", Default: "warning", Type: "choice", Options: []string{"info", "warning", "debug"}},
		{Name: "quoted", Default: "  spaced  ", Type: "string"},
		{Name: "multiline", Description: "First line\nsecond line\n", Type: "string"},
		{Name: "bare", Type: "string"},
		{Name: "numeric_options", Type: "choice", Options: []string{"1", "2", "3"}},
	}, inputs)
}

func TestParseWorkflowDispatchInputsErrors(t *testing.T) {
	scenarios := []struct {
		name    string
		content string
	}{
		{
			name:    "malformed YAML",
			content: "on:\n  workflow_dispatch:\n    inputs:\n      x: [unterminated\n",
		},
		{
			name:    "tab indentation",
			content: "on:\n\tworkflow_dispatch:\n",
		},
		{
			name:    "input that isn't a mapping",
			content: "on:\n  workflow_dispatch:\n    inputs:\n      x: just a string\n",
		},
		{
			name:    "required that isn't a boolean",
			content: "on:\n  workflow_dispatch:\n    inputs:\n      x:\n        required: maybe\n",
		},
		{
			name:    "options that aren't a list",
			content: "on:\n  workflow_dispatch:\n    inputs:\n      x:\n        type: choice\n        options: a\n",
		},
		{
			name:    "default that isn't a scalar",
			content: "on:\n  workflow_dispatch:\n    inputs:\n      x:\n        default: [a, b]\n",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			inputs, err := ParseWorkflowDispatchInputs(s.content)
			assert.Error(t, err)
			assert.NotErrorIs(t, err, ErrNotDispatchable)
			assert.Nil(t, inputs)
		})
	}
}
