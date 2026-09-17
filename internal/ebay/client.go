package ebay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const browseBase = "https://api.ebay.com/buy/browse/v1"

var Markets = map[string]string{
	"de": "EBAY_DE",
	"es": "EBAY_ES",
	"it": "EBAY_IT",
	"fr": "EBAY_FR",
	"gb": "EBAY_GB",
	"uk": "EBAY_GB",
	"ie": "EBAY_IE",
	"nl": "EBAY_NL",
	"be": "EBAY_BE",
	"at": "EBAY_AT",
	"ch": "EBAY_CH",
	"pl": "EBAY_PL",
	"us": "EBAY_US",
	"ca": "EBAY_CA",
	"au": "EBAY_AU",
}

var marketLocale = map[string]string{
	"EBAY_DE": "de-DE",
	"EBAY_ES": "es-ES",
	"EBAY_IT": "it-IT",
	"EBAY_FR": "fr-FR",
	"EBAY_GB": "en-GB",
	"EBAY_IE": "en-IE",
	"EBAY_NL": "nl-NL",
	"EBAY_BE": "nl-BE",
	"EBAY_AT": "de-AT",
	"EBAY_CH": "de-CH",
	"EBAY_PL": "pl-PL",
	"EBAY_US": "en-US",
	"EBAY_CA": "en-CA",
	"EBAY_AU": "en-AU",
}

// ResolveMarket accepts either a country shorthand ("de") or a full marketplace
// id ("EBAY_DE").
func ResolveMarket(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("no marketplace given")
	}
	if strings.HasPrefix(strings.ToUpper(s), "EBAY_") {
		return strings.ToUpper(s), nil
	}
	if m, ok := Markets[strings.ToLower(s)]; ok {
		return m, nil
	}
	return "", fmt.Errorf("unknown marketplace %q, try `ebay markets`", s)
}

type Client struct {
	HTTP  *http.Client
	Creds Credentials
	cache tokenCache
}

func New(creds Credentials) *Client {
	return &Client{
		HTTP:  &http.Client{Timeout: 30 * time.Second},
		Creds: creds,
	}
}

func (c *Client) do(ctx context.Context, path string, q url.Values, market, shipTo string, out any) error {
	token, err := c.cache.get(ctx, c.HTTP, c.Creds)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, browseBase+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", market)
	if loc, ok := marketLocale[market]; ok {
		req.Header.Set("Accept-Language", loc)
	}
	if shipTo != "" {
		// Without this eBay quotes shipping to the marketplace's own country,
		// which is the wrong number for a cross-border buyer.
		req.Header.Set("X-EBAY-C-ENDUSERCTX", "contextualLocation="+url.QueryEscape("country="+strings.ToUpper(shipTo)))
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("eBay returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
