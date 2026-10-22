package github_commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Finding pull request templates the way the GitHub CLI does (cli/cli's
// pkg/githubtemplate), which is the way GitHub does:
// https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/creating-a-pull-request-template-for-your-repository

// PullRequestTemplate is a template for the description of a new pull
// request.
type PullRequestTemplate struct {
	// from the template's front matter, or its file name
	Name string
	Body string
}

// FindPullRequestTemplates returns the pull request templates of the working
// tree at rootDir: those in a PULL_REQUEST_TEMPLATE directory if there is
// one, or otherwise the single PULL_REQUEST_TEMPLATE file, if there is one.
func FindPullRequestTemplates(rootDir string) []PullRequestTemplate {
	paths := findTemplatesInDir(rootDir, "PULL_REQUEST_TEMPLATE")
	if len(paths) == 0 {
		if path := findLegacyTemplate(rootDir, "PULL_REQUEST_TEMPLATE"); path != "" {
			paths = []string{path}
		}
	}

	templates := []PullRequestTemplate{}
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		templates = append(templates, PullRequestTemplate{
			Name: templateName(path, contents),
			Body: string(templateBody(contents)),
		})
	}
	return templates
}

func templateCandidateDirs(rootDir string) []string {
	return []string{filepath.Join(rootDir, ".github"), rootDir, filepath.Join(rootDir, "docs")}
}

func findTemplatesInDir(rootDir string, name string) []string {
	results := []string{}

mainLoop:
	for _, dir := range templateCandidateDirs(rootDir) {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if strings.EqualFold(file.Name(), name) && file.IsDir() {
				templates, err := os.ReadDir(filepath.Join(dir, file.Name()))
				if err != nil {
					break
				}
				for _, template := range templates {
					if strings.HasSuffix(template.Name(), ".md") && template.Type() != fs.ModeSymlink {
						results = append(results, filepath.Join(dir, file.Name(), template.Name()))
					}
				}
				if len(results) > 0 {
					break mainLoop
				}
				break
			}
		}
	}

	sort.Strings(results)
	return results
}

func findLegacyTemplate(rootDir string, name string) string {
	namePattern := regexp.MustCompile(fmt.Sprintf(`(?i)^%s(\.|$)`, strings.ReplaceAll(name, "_", "[_-]")))

	for _, dir := range templateCandidateDirs(rootDir) {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if namePattern.MatchString(file.Name()) && !file.IsDir() && file.Type() != fs.ModeSymlink {
				return filepath.Join(dir, file.Name())
			}
		}
	}
	return ""
}

var frontMatterDelimiter = regexp.MustCompile(`(?m)^---\r?\n(\s*\r?\n)?`)

// frontMatterEnd returns where a template's YAML front matter ends, or -1 if
// it has none.
func frontMatterEnd(contents []byte) int {
	matches := frontMatterDelimiter.FindAllIndex(contents, 2)
	if len(matches) > 1 && matches[0][0] == 0 {
		return matches[1][1]
	}
	return -1
}

func templateName(path string, contents []byte) string {
	if end := frontMatterEnd(contents); end >= 0 {
		var frontMatter struct {
			Name string `yaml:"name"`
		}
		if yaml.Unmarshal(contents[:end], &frontMatter) == nil && frontMatter.Name != "" {
			return frontMatter.Name
		}
	}
	return filepath.Base(path)
}

func templateBody(contents []byte) []byte {
	if end := frontMatterEnd(contents); end >= 0 {
		return contents[end:]
	}
	return contents
}
