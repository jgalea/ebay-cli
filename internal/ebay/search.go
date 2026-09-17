package ebay

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Query struct {
	Keywords string
	Market   string
	ShipTo   string
	Limit    int
	Offset   int
	Sort     string

	MinPrice string
	MaxPrice string
	Currency string

	Conditions      []string
	SellerLoc       string
	IncludeAuctions bool
	FixedOnly       bool
}

var sortAliases = map[string]string{
	"price":   "price",
	"cheap":   "price",
	"-price":  "-price",
	"dear":    "-price",
	"new":     "newlyListed",
	"newest":  "newlyListed",
	"ending":  "endingSoonest",
	"soonest": "endingSoonest",
}

func resolveSort(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if v, ok := sortAliases[strings.ToLower(s)]; ok {
		return v, nil
	}
	return "", fmt.Errorf("unknown sort %q (price, -price, new, ending)", s)
}

// buildFilter assembles eBay's comma-separated filter string. Price filters need
// priceCurrency alongside them or eBay rejects the request.
func buildFilter(q Query) (string, error) {
	var parts []string

	if q.MinPrice != "" || q.MaxPrice != "" {
		if q.Currency == "" {
			return "", fmt.Errorf("a price filter needs a currency")
		}
		parts = append(parts, "price:["+q.MinPrice+".."+q.MaxPrice+"]")
		parts = append(parts, "priceCurrency:"+q.Currency)
	}
	if len(q.Conditions) > 0 {
		parts = append(parts, "conditions:{"+strings.Join(q.Conditions, "|")+"}")
	}
	if q.SellerLoc != "" {
		parts = append(parts, "itemLocationCountry:"+strings.ToUpper(q.SellerLoc))
	}

	// eBay hides auction-only listings unless buyingOptions says otherwise, so an
	// item that has attracted bids silently disappears from a default search.
	switch {
	case q.FixedOnly:
		parts = append(parts, "buyingOptions:{FIXED_PRICE}")
	case q.IncludeAuctions:
		parts = append(parts, "buyingOptions:{AUCTION|FIXED_PRICE}")
	}

	return strings.Join(parts, ","), nil
}

func (c *Client) Search(ctx context.Context, q Query) (*SearchResult, error) {
	if strings.TrimSpace(q.Keywords) == "" {
		return nil, fmt.Errorf("no search keywords given")
	}
	market, err := ResolveMarket(q.Market)
	if err != nil {
		return nil, err
	}
	sort, err := resolveSort(q.Sort)
	if err != nil {
		return nil, err
	}
	filter, err := buildFilter(q)
	if err != nil {
		return nil, err
	}

	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	v := url.Values{}
	v.Set("q", q.Keywords)
	v.Set("limit", strconv.Itoa(q.Limit))
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	if sort != "" {
		v.Set("sort", sort)
	}
	if filter != "" {
		v.Set("filter", filter)
	}

	var out SearchResult
	if err := c.do(ctx, "/item_summary/search", v, market, q.ShipTo, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
