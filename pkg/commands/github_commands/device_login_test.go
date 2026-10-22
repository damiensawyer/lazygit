package github_commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type formRequest struct {
	method      string
	path        string
	contentType string
	accept      string
	form        url.Values
}

// newLoginServer starts a fake GitHub host for the device login, and returns
// the host name to pass to the login functions. The login functions only
// talk https to the host they're given, so the fake server uses TLS, and the
// default transport is swapped for one that trusts its certificate while the
// test runs.
func newLoginServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (string, func() []formRequest) {
	t.Helper()

	var mutex sync.Mutex
	requests := []formRequest{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		mutex.Lock()
		requests = append(requests, formRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			accept:      r.Header.Get("Accept"),
			form:        r.PostForm,
		})
		mutex.Unlock()
		handler(w, r)
	}))

	originalTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
		server.Close()
	})

	return server.Listener.Addr().String(), func() []formRequest {
		mutex.Lock()
		defer mutex.Unlock()
		return append([]formRequest{}, requests...)
	}
}

func respondWith(status int, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestRequestDeviceCode(t *testing.T) {
	host, requests := newLoginServer(t, respondWith(http.StatusOK, `{
		"device_code": "3584d83530557fdd1f46af8289938c8ef79f9dc5",
		"user_code": "WDJB-MJHT",
		"verification_uri": "https://github.com/login/device",
		"expires_in": 900,
		"interval": 5
	}`))

	code, err := RequestDeviceCode(context.Background(), host)
	assert.NoError(t, err)

	assert.Equal(t, &DeviceCode{
		DeviceCode:      "3584d83530557fdd1f46af8289938c8ef79f9dc5",
		UserCode:        "WDJB-MJHT",
		VerificationURI: "https://github.com/login/device",
		ExpiresIn:       900,
		Interval:        5,
	}, code)
	assert.Equal(t, []formRequest{{
		method:      "POST",
		path:        "/login/device/code",
		contentType: "application/x-www-form-urlencoded",
		accept:      "application/json",
		form:        url.Values{"client_id": {oauthClientID}, "scope": {"repo,read:org,workflow"}},
	}}, requests())
}

func TestRequestDeviceCodeFailures(t *testing.T) {
	scenarios := []struct {
		name          string
		status        int
		response      string
		expectedError string
	}{
		{
			name:          "HTTP error",
			status:        http.StatusServiceUnavailable,
			response:      `{"error": "unavailable"}`,
			expectedError: "HTTP 503",
		},
		{
			name:          "not found",
			status:        http.StatusNotFound,
			response:      `Not Found`,
			expectedError: "HTTP 404",
		},
		{
			name:          "error instead of a code",
			status:        http.StatusOK,
			response:      `{"error": "unauthorized_client", "error_description": "Device flow is disabled"}`,
			expectedError: "unexpected response from GitHub",
		},
		{
			name:          "no user code",
			status:        http.StatusOK,
			response:      `{"device_code": "dc", "verification_uri": "https://github.com/login/device"}`,
			expectedError: "unexpected response from GitHub",
		},
		{
			name:          "no device code",
			status:        http.StatusOK,
			response:      `{"user_code": "WDJB-MJHT", "verification_uri": "https://github.com/login/device"}`,
			expectedError: "unexpected response from GitHub",
		},
		{
			name:          "no verification URI",
			status:        http.StatusOK,
			response:      `{"device_code": "dc", "user_code": "WDJB-MJHT"}`,
			expectedError: "unexpected response from GitHub",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			host, _ := newLoginServer(t, respondWith(s.status, s.response))

			code, err := RequestDeviceCode(context.Background(), host)

			assert.EqualError(t, err, s.expectedError)
			assert.Nil(t, code)
		})
	}
}

func TestRequestDeviceCodeWithMalformedJSON(t *testing.T) {
	host, _ := newLoginServer(t, respondWith(http.StatusOK, `device_code=dc&user_code=WDJB-MJHT`))

	code, err := RequestDeviceCode(context.Background(), host)

	assert.Error(t, err)
	assert.Nil(t, code)
}

func TestPollDeviceToken(t *testing.T) {
	scenarios := []struct {
		name          string
		status        int
		response      string
		expectedToken string
		expectedError error
		expectedText  string
	}{
		{
			name:          "token",
			response:      `{"access_token": "gho_abc", "token_type": "bearer", "scope": "repo,read:org,workflow"}`,
			expectedToken: "gho_abc",
		},
		{
			name:          "authorization pending",
			response:      `{"error": "authorization_pending", "error_description": "The authorization request is still pending."}`,
			expectedError: errAuthorizationPending,
		},
		{
			name:          "slow down",
			response:      `{"error": "slow_down", "interval": 10}`,
			expectedError: errSlowDown,
		},
		{
			name:         "expired token",
			response:     `{"error": "expired_token"}`,
			expectedText: "the code expired before the login was completed",
		},
		{
			name:         "access denied",
			response:     `{"error": "access_denied"}`,
			expectedText: "access was denied",
		},
		{
			name:         "other error",
			response:     `{"error": "incorrect_device_code"}`,
			expectedText: "incorrect_device_code",
		},
		{
			name:         "neither token nor error",
			response:     `{}`,
			expectedText: "GitHub sent no token",
		},
		{
			name:         "HTTP error",
			status:       http.StatusInternalServerError,
			response:     `{"access_token": "gho_abc"}`,
			expectedText: "HTTP 500",
		},
		{
			name:     "authorization pending, sent as a bad request",
			status:   http.StatusBadRequest,
			response: `{"error": "authorization_pending"}`,

			expectedError: errAuthorizationPending,
		},
		{
			name:     "slow down, sent as a bad request",
			status:   http.StatusBadRequest,
			response: `{"error": "slow_down", "interval": 10}`,

			expectedError: errSlowDown,
		},
		{
			name:     "expired token, sent as a bad request",
			status:   http.StatusBadRequest,
			response: `{"error": "expired_token"}`,

			expectedText: "the code expired before the login was completed",
		},
		{
			name:     "access denied, sent as a bad request",
			status:   http.StatusBadRequest,
			response: `{"error": "access_denied"}`,

			expectedText: "access was denied",
		},
		{
			name:         "bad request without an error",
			status:       http.StatusBadRequest,
			response:     `Bad Request`,
			expectedText: "HTTP 400",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			status := s.status
			if status == 0 {
				status = http.StatusOK
			}
			host, requests := newLoginServer(t, respondWith(status, s.response))

			token, err := pollDeviceToken(context.Background(), host, "the-device-code")

			assert.Equal(t, s.expectedToken, token)
			switch {
			case s.expectedError != nil:
				assert.ErrorIs(t, err, s.expectedError)
			case s.expectedText != "":
				assert.EqualError(t, err, s.expectedText)
			default:
				assert.NoError(t, err)
			}
			assert.Equal(t, []formRequest{{
				method:      "POST",
				path:        "/login/oauth/access_token",
				contentType: "application/x-www-form-urlencoded",
				accept:      "application/json",
				form: url.Values{
					"client_id":   {oauthClientID},
					"device_code": {"the-device-code"},
					"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
				},
			}}, requests())
		})
	}
}

func TestAwaitDeviceToken(t *testing.T) {
	scenarios := []struct {
		name string
		// what the server answers to each poll in turn
		responses     []string
		cancelled     bool
		interval      int
		expiresIn     int
		expectedToken string
		expectedError string
		expectedPolls int
	}{
		{
			name:          "keeps polling while authorization is pending",
			responses:     []string{`{"error": "authorization_pending"}`, `{"access_token": "gho_abc"}`},
			interval:      1,
			expiresIn:     30,
			expectedToken: "gho_abc",
			expectedPolls: 2,
		},
		{
			name:          "gives up when access is denied",
			responses:     []string{`{"error": "access_denied"}`},
			interval:      1,
			expiresIn:     30,
			expectedError: "access was denied",
			expectedPolls: 1,
		},
		{
			// After slow_down the interval grows from 1 to 6 seconds, so the
			// code expires before the second poll.
			name:          "waits longer when asked to slow down",
			responses:     []string{`{"error": "slow_down"}`, `{"access_token": "too late"}`},
			interval:      1,
			expiresIn:     3,
			expectedError: "the code expired before the login was completed",
			expectedPolls: 1,
		},
		{
			// With no interval given, the first poll would be after 5 seconds.
			name:          "gives up when the code expires",
			expiresIn:     1,
			expectedError: "the code expired before the login was completed",
			expectedPolls: 0,
		},
		{
			name:          "stops when cancelled",
			cancelled:     true,
			interval:      1,
			expiresIn:     30,
			expectedError: context.Canceled.Error(),
			expectedPolls: 0,
		},
	}

	// The polling interval is in whole seconds, so the scenarios run
	// concurrently against one server, which tells them apart by their
	// device codes: the index of the scenario.
	var mutex sync.Mutex
	polls := map[string]int{}
	host, _ := newLoginServer(t, func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		deviceCode := r.PostForm.Get("device_code")
		index, _ := strconv.Atoi(deviceCode)
		_, _ = w.Write([]byte(scenarios[index].responses[polls[deviceCode]]))
		polls[deviceCode]++
	})

	type result struct {
		token    string
		err      error
		duration time.Duration
	}
	results := make([]result, len(scenarios))
	var wg sync.WaitGroup
	for i, s := range scenarios {
		wg.Go(func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if s.cancelled {
				cancel()
			}
			start := time.Now()
			code := &DeviceCode{DeviceCode: strconv.Itoa(i), Interval: s.interval, ExpiresIn: s.expiresIn}
			token, err := AwaitDeviceToken(ctx, host, code)
			results[i] = result{token: token, err: err, duration: time.Since(start)}
		})
	}
	wg.Wait()

	for i, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expectedToken, results[i].token)
			if s.expectedError == "" {
				assert.NoError(t, results[i].err)
			} else {
				assert.EqualError(t, results[i].err, s.expectedError)
			}
			assert.Equal(t, s.expectedPolls, polls[strconv.Itoa(i)])
			if s.expiresIn > 0 {
				assert.Less(t, results[i].duration, time.Duration(s.expiresIn+1)*time.Second)
			}
		})
	}
}

func TestFetchUsername(t *testing.T) {
	var auth, path string
	host, _ := newLoginServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"login": "octocat", "id": 1}`))
	})

	username, err := FetchUsername(context.Background(), host, "gho_abc")
	assert.NoError(t, err)

	assert.Equal(t, "octocat", username)
	assert.Equal(t, "Bearer gho_abc", auth)
	// A host other than github.com counts as GitHub Enterprise Server.
	assert.Equal(t, "/api/v3/user", path)
}

func TestFetchUsernameWithInvalidToken(t *testing.T) {
	host, _ := newLoginServer(t, respondWith(http.StatusUnauthorized, `{"message": "Bad credentials"}`))

	username, err := FetchUsername(context.Background(), host, "gho_bad")

	assert.EqualError(t, err, "GitHub API request failed: HTTP 401: Bad credentials")
	assert.Empty(t, username)
}
