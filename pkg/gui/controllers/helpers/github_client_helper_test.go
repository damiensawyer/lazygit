package helpers

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubTokens(t *testing.T) {
	var tokens GithubTokens
	assert.Empty(t, tokens.get("github.com"))

	tokens.set("github.com", "gho_one")
	tokens.set("git.acme.com", "ghe_two")
	assert.Equal(t, "gho_one", tokens.get("github.com"))
	assert.Equal(t, "ghe_two", tokens.get("git.acme.com"))
	assert.Empty(t, tokens.get("GitHub.com"))

	tokens.set("github.com", "gho_three")
	assert.Equal(t, "gho_three", tokens.get("github.com"))

	// Forgetting a token that turned out to be invalid.
	tokens.set("github.com", "")
	assert.Empty(t, tokens.get("github.com"))
	assert.Equal(t, "ghe_two", tokens.get("git.acme.com"))
}

func TestGithubTokensConcurrentAccess(t *testing.T) {
	var tokens GithubTokens
	var wg sync.WaitGroup
	for i := range 20 {
		host := fmt.Sprintf("host%d", i%4)
		wg.Go(func() {
			tokens.set(host, "token")
			_ = tokens.get(host)
		})
	}
	wg.Wait()

	for i := range 4 {
		assert.Equal(t, "token", tokens.get(fmt.Sprintf("host%d", i)))
	}
}
