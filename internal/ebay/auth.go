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
	"path/filepath"
	"runtime"
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
// touching a secret store. After that it tries the macOS Keychain, then the
// credentials file, which is the only store on Linux and Windows.
func LoadCredentials() (Credentials, error) {
	c := Credentials{os.Getenv("EBAY_CLIENT_ID"), os.Getenv("EBAY_CLIENT_SECRET")}
	if runtime.GOOS == "darwin" {
		if c.ClientID == "" {
			c.ClientID = keychainRead(keychainClientID)
		}
		if c.ClientSecret == "" {
			c.ClientSecret = keychainRead(keychainClientSecret)
		}
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		path, err := CredentialsPath()
		if err != nil {
			return Credentials{}, err
		}
		if b, err := os.ReadFile(path); err == nil {
			var f credentialsFile
			if err := json.Unmarshal(b, &f); err != nil {
				return Credentials{}, fmt.Errorf("%s is not readable as JSON: %w", path, err)
			}
			if c.ClientID == "" {
				c.ClientID = f.ClientID
			}
			if c.ClientSecret == "" {
				c.ClientSecret = f.ClientSecret
			}
		}
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return Credentials{}, fmt.Errorf("no credentials found, run `ebay auth` (or set EBAY_CLIENT_ID and EBAY_CLIENT_SECRET)")
	}
	return c, nil
}

type credentialsFile struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// CredentialsPath is the file-based credential store, next to the watches.
func CredentialsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ebay-cli", "credentials.json"), nil
}

func keychainRead(service string) string {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
