package ebay

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Image struct {
	ImageURL string `json:"imageUrl"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ItemDetail is the full listing as returned by the Browse API's getItem.
type ItemDetail struct {
	ItemID               string           `json:"itemId"`
	LegacyItemID         string           `json:"legacyItemId"`
	Title                string           `json:"title"`
	Subtitle             string           `json:"subtitle,omitempty"`
	ShortDescription     string           `json:"shortDescription,omitempty"`
	Description          string           `json:"description,omitempty"`
	Condition            string           `json:"condition"`
	ConditionID          string           `json:"conditionId"`
	ConditionDescription string           `json:"conditionDescription,omitempty"`
	Price                *Money           `json:"price"`
	CurrentBidPrice      *Money           `json:"currentBidPrice"`
	BidCount             int              `json:"bidCount"`
	BuyingOptions        []string         `json:"buyingOptions"`
	ItemWebURL           string           `json:"itemWebUrl"`
	ItemLocation         *Location        `json:"itemLocation"`
	Seller               *Seller          `json:"seller"`
	ShippingOptions      []ShippingOption `json:"shippingOptions"`
	ItemEndDate          *time.Time       `json:"itemEndDate"`
	Image                *Image           `json:"image"`
	AdditionalImages     []Image          `json:"additionalImages"`
	LocalizedAspects     []NameValue      `json:"localizedAspects"`
	CategoryPath         string           `json:"categoryPath"`
}

// Summary converts the detail to the summary type so price and shipping helpers
// can be shared with search results.
func (d ItemDetail) Summary() Item {
	return Item{
		ItemID:          d.ItemID,
		Title:           d.Title,
		Condition:       d.Condition,
		Price:           d.Price,
		CurrentBidPrice: d.CurrentBidPrice,
		BidCount:        d.BidCount,
		BuyingOptions:   d.BuyingOptions,
		ItemWebURL:      d.ItemWebURL,
		ItemLocation:    d.ItemLocation,
		Seller:          d.Seller,
		ShippingOptions: d.ShippingOptions,
		ItemEndDate:     d.ItemEndDate,
	}
}

// ImageURLs returns the listing's gallery pictures followed by any pictures the
// seller embedded in the description, without duplicates. Sellers often show
// SMART data or test results only as a screenshot in the description.
func (d ItemDetail) ImageURLs() []string {
	var out []string
	seen := map[string]bool{}
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	if d.Image != nil {
		add(d.Image.ImageURL)
	}
	for _, im := range d.AdditionalImages {
		add(im.ImageURL)
	}
	for _, u := range DescriptionImages(d.Description) {
		add(u)
	}
	return out
}

var (
	itmPath   = regexp.MustCompile(`/itm/(?:[^/?#]+/)?(\d{9,15})`)
	legacyID  = regexp.MustCompile(`^\d{9,15}$`)
	restID    = regexp.MustCompile(`^v1\|\d+\|\d+$`)
	imgSrc    = regexp.MustCompile(`(?is)<img[^>]+src\s*=\s*["']([^"']+)["']`)
	scriptTag = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	breakTag  = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/li|/tr|/h[1-6]|/table|hr)[^>]*>`)
	cellTag   = regexp.MustCompile(`(?i)<\s*/t[dh]\s*>`)
	liTag     = regexp.MustCompile(`(?i)<\s*li[^>]*>`)
	anyTag    = regexp.MustCompile(`(?s)<[^>]*>`)
	spaces    = regexp.MustCompile(`[ \t\x{00a0}]+`)
	blankRuns = regexp.MustCompile(`\n{3,}`)
)

// ParseItemRef accepts a Browse API id (v1|123|0), a legacy item number, or a
// listing URL, and tells which of the first two it resolved to.
func ParseItemRef(s string) (id string, legacy bool, err error) {
	s = strings.TrimSpace(s)
	switch {
	case restID.MatchString(s):
		return s, false, nil
	case legacyID.MatchString(s):
		return s, true, nil
	}
	if m := itmPath.FindStringSubmatch(s); m != nil {
		return m[1], true, nil
	}
	return "", false, fmt.Errorf("%q is not an item number, a v1|…|… id or an eBay listing URL", s)
}

// GetItem fetches the full listing including the seller's description.
func (c *Client) GetItem(ctx context.Context, ref, market, shipTo string) (*ItemDetail, error) {
	id, legacy, err := ParseItemRef(ref)
	if err != nil {
		return nil, err
	}
	m, err := ResolveMarket(market)
	if err != nil {
		return nil, err
	}
	var out ItemDetail
	if legacy {
		v := url.Values{}
		v.Set("legacy_item_id", id)
		err = c.do(ctx, "/item/get_item_by_legacy_id", v, m, shipTo, &out)
	} else {
		err = c.do(ctx, "/item/"+url.PathEscape(id), url.Values{}, m, shipTo, &out)
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DescriptionImages returns the src of every <img> in a description.
func DescriptionImages(desc string) []string {
	var out []string
	for _, m := range imgSrc.FindAllStringSubmatch(desc, -1) {
		u := html.UnescapeString(m[1])
		if strings.HasPrefix(u, "//") {
			u = "https:" + u
		}
		if strings.HasPrefix(u, "http") {
			out = append(out, u)
		}
	}
	return out
}

// HTMLToText turns a listing description into readable plain text, keeping
// line structure but dropping markup, scripts and styles.
func HTMLToText(s string) string {
	s = scriptTag.ReplaceAllString(s, "")
	s = liTag.ReplaceAllString(s, "\n- ")
	s = cellTag.ReplaceAllString(s, " | ")
	s = breakTag.ReplaceAllString(s, "\n")
	s = anyTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r", "")
	s = spaces.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l), "|"))
	}
	s = strings.Join(lines, "\n")
	s = blankRuns.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
