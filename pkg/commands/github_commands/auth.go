package github_commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
	ghconfig "github.com/cli/go-gh/v2/pkg/config"
	"gopkg.in/yaml.v3"
)

const (
	// gh's OAuth app. Using it means the consent page says "GitHub CLI", and
	// the token works for gh too, which shares where we store it.
	oauthClientID = "178c6fc778ccc68e1d6a"

	// repo covers Actions data of private repositories; workflow is needed to
	// re-run and dispatch workflows; read:org lets runs of repositories owned
	// by the user's organizations be read.
	oauthScopes = "repo,read:org,workflow"
)

var (
	errAuthorizationPending = errors.New("authorization pending")
	errSlowDown             = errors.New("slow down")
)

// DeviceCode is what GitHub hands out to start a device login: the user enters
// UserCode at VerificationURI while we poll for the resulting token.
type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// SupportsDeviceLogin reports whether the device login flow is available for
// a host. It isn't on GitHub Enterprise, where gh's OAuth app doesn't exist.
func SupportsDeviceLogin(host string) bool {
	return !auth.IsEnterprise(host)
}

func postForm(ctx context.Context, u string, form url.Values, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// RequestDeviceCode starts a device login.
func RequestDeviceCode(ctx context.Context, host string) (*DeviceCode, error) {
	var code DeviceCode
	err := postForm(ctx, fmt.Sprintf("https://%s/login/device/code", host),
		url.Values{"client_id": {oauthClientID}, "scope": {oauthScopes}}, &code)
	if err != nil {
		return nil, err
	}
	if code.DeviceCode == "" || code.UserCode == "" || code.VerificationURI == "" {
		return nil, errors.New("unexpected response from GitHub")
	}
	return &code, nil
}

// AwaitDeviceToken polls GitHub until the user has completed the device login
// in their browser, and returns the token it produced. It gives up when the
// code expires, the user denies access, or ctx is cancelled.
func AwaitDeviceToken(ctx context.Context, host string, code *DeviceCode) (string, error) {
	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	expiresIn := time.Duration(code.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 15 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, expiresIn)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return "", errors.New("the code expired before the login was completed")
			}
			return "", ctx.Err()
		case <-time.After(interval):
		}

		token, err := pollDeviceToken(ctx, host, code.DeviceCode)
		switch {
		case err == nil:
			return token, nil
		case errors.Is(err, errAuthorizationPending):
		case errors.Is(err, errSlowDown):
			interval += 5 * time.Second
		default:
			return "", err
		}
	}
}

func pollDeviceToken(ctx context.Context, host string, deviceCode string) (string, error) {
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	err := postForm(ctx, fmt.Sprintf("https://%s/login/oauth/access_token", host), url.Values{
		"client_id":   {oauthClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}, &result)
	if err != nil {
		return "", err
	}

	switch result.Error {
	case "":
		if result.AccessToken == "" {
			return "", errors.New("GitHub sent no token")
		}
		return result.AccessToken, nil
	case "authorization_pending":
		return "", errAuthorizationPending
	case "slow_down":
		return "", errSlowDown
	case "expired_token":
		return "", errors.New("the code expired before the login was completed")
	case "access_denied":
		return "", errors.New("access was denied")
	default:
		return "", errors.New(result.Error)
	}
}

// FetchUsername returns the login of the user a token belongs to, failing if
// the token isn't valid.
func FetchUsername(ctx context.Context, host string, token string) (string, error) {
	client := NewClient(Repo{Host: host, Token: token})
	var user struct {
		Login string `json:"login"`
	}
	if err := client.getJSON(ctx, "/user", nil, &user); err != nil {
		return "", err
	}
	return user.Login, nil
}

// StoreToken saves a token in the GitHub CLI's hosts.yml, the same way
// `gh auth login --insecure-storage` does, so that it is found next time,
// both by us and by gh.
func StoreToken(host string, username string, token string) error {
	path := filepath.Join(ghconfig.ConfigDir(), "hosts.yml")

	// Read the file afresh rather than through go-gh's config package, which
	// keeps the first version it read for the rest of the process: writing
	// that back would undo whatever gh changed in the file since.
	var doc yaml.Node
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(bytes.TrimSpace(content)) > 0 {
		if err := yaml.Unmarshal(content, &doc); err != nil {
			return fmt.Errorf("couldn't parse %s: %w", path, err)
		}
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}

	hostNode := mappingEntry(doc.Content[0], host)
	setScalar(hostNode, "oauth_token", token)
	setScalar(hostNode, "user", username)
	if mappingValue(hostNode, "git_protocol") == nil {
		setScalar(hostNode, "git_protocol", "https")
	}
	setScalar(mappingEntry(mappingEntry(hostNode, "users"), username), "oauth_token", token)

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(4)
	if err := encoder.Encode(&doc); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, out.Bytes(), 0o600)
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// mappingEntry returns the mapping stored under key, creating it if needed.
func mappingEntry(mapping *yaml.Node, key string) *yaml.Node {
	if value := mappingValue(mapping, key); value != nil {
		if value.Kind != yaml.MappingNode {
			*value = yaml.Node{Kind: yaml.MappingNode}
		}
		return value
	}
	value := &yaml.Node{Kind: yaml.MappingNode}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
	return value
}

func setScalar(mapping *yaml.Node, key string, value string) {
	if existing := mappingValue(mapping, key); existing != nil {
		*existing = yaml.Node{Kind: yaml.ScalarNode, Value: value}
		return
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value})
}
