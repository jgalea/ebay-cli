package ebay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	tokenURL = "https://api.ebay.com/identity/v1/oauth2/token"
	apiScope = "https://api.ebay.com/oauth/api_scope"

	keychainClientID     = "claude-ebay-client-id"
	keychainClientSecret = "claude-ebay-client-secret"
)

type Credentials struct {
	ClientID     string
	ClientSecret string
}

// LoadCredentials prefers the environment so CI and one-off runs work without
// touching the Keychain.
func LoadCredentials() (Credentials, error) {
	id := os.Getenv("EBAY_CLIENT_ID")
	secret := os.Getenv("EBAY_CLIENT_SECRET")
	if id != "" && secret != "" {
		return Credentials{id, secret}, nil
	}
	var err error
	if id == "" {
		if id, err = keychainRead(keychainClientID); err != nil {
			return Credentials{}, err
		}
	}
	if secret == "" {
		if secret, err = keychainRead(keychainClientSecret); err != nil {
			return Credentials{}, err
		}
	}
	return Credentials{id, secret}, nil
}

func keychainRead(service string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		return "", fmt.Errorf("no credentials: keychain item %q not found, run `ebay auth` (or set EBAY_CLIENT_ID and EBAY_CLIENT_SECRET)", service)
	}
	return strings.TrimSpace(string(out)), nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

type tokenCache struct {
	mu    sync.Mutex
	token string
	until time.Time
}

func (c *tokenCache) get(ctx context.Context, hc *http.Client, creds Credentials) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.until) {
		return c.token, nil
	}

	body := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {apiScope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(creds.ClientID + ":" + creds.ClientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basic)

	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("token endpoint returned %s and an unreadable body", resp.Status)
	}
	if resp.StatusCode != http.StatusOK || tr.AccessToken == "" {
		if tr.ErrorDesc != "" {
			return "", fmt.Errorf("token request failed (%s): %s", resp.Status, tr.ErrorDesc)
		}
		return "", fmt.Errorf("token request failed: %s", resp.Status)
	}

	c.token = tr.AccessToken
	// Retire the token a minute early so a long run never uses one mid-expiry.
	c.until = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - time.Minute)
	return c.token, nil
}
