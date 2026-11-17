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
	ConditionIDs    []string
	CategoryIDs     []string
	Aspects         []Aspect
	SellerLoc       string
	IncludeAuctions bool
	FixedOnly       bool
}

// NotDefectiveConditionIDs is every eBay condition except 7000, "for parts or
// not working".
var NotDefectiveConditionIDs = []string{
	"1000", "1500", "1750", "2000", "2010", "2020", "2030",
	"2500", "2750", "3000", "4000", "5000", "6000",
}

// Aspect is an item specific to filter on, such as Speicherkapazität with the
// values 2TB, 3 TB and 4 TB. Names and values must match the marketplace's own
// spelling exactly; `ebay aspects` lists them.
type Aspect struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// ParseAspect reads the command-line form "Name=value1|value2".
func ParseAspect(s string) (Aspect, error) {
	name, values, ok := strings.Cut(s, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return Aspect{}, fmt.Errorf("aspect %q must look like Name=value1|value2", s)
	}
	a := Aspect{Name: name}
	for _, v := range strings.Split(values, "|") {
		if v = strings.TrimSpace(v); v != "" {
			a.Values = append(a.Values, v)
		}
	}
	if len(a.Values) == 0 {
		return Aspect{}, fmt.Errorf("aspect %q has no values", s)
	}
	return a, nil
}

func (a Aspect) String() string { return a.Name + "=" + strings.Join(a.Values, "|") }

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
	if len(q.ConditionIDs) > 0 {
		parts = append(parts, "conditionIds:{"+strings.Join(q.ConditionIDs, "|")+"}")
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

// buildAspectFilter assembles eBay's aspect_filter. eBay scopes it to exactly
// one category, so the first --category is used.
func buildAspectFilter(q Query) (string, error) {
	if len(q.Aspects) == 0 {
		return "", nil
	}
	if len(q.CategoryIDs) == 0 {
		return "", fmt.Errorf("aspect filters need a category (--category)")
	}
	parts := []string{"categoryId:" + q.CategoryIDs[0]}
	for _, a := range q.Aspects {
		parts = append(parts, a.Name+":{"+strings.Join(a.Values, "|")+"}")
	}
	return strings.Join(parts, ","), nil
}

func (c *Client) Search(ctx context.Context, q Query) (*SearchResult, error) {
	v, market, err := searchParams(q)
	if err != nil {
		return nil, err
	}
	var out SearchResult
	if err := c.do(ctx, "/item_summary/search", v, market, q.ShipTo, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func searchParams(q Query) (url.Values, string, error) {
	keywords := strings.TrimSpace(q.Keywords)
	if keywords == "" && len(q.CategoryIDs) == 0 {
		return nil, "", fmt.Errorf("give search keywords, a category, or both")
	}
	market, err := ResolveMarket(q.Market)
	if err != nil {
		return nil, "", err
	}
	sort, err := resolveSort(q.Sort)
	if err != nil {
		return nil, "", err
	}
	filter, err := buildFilter(q)
	if err != nil {
		return nil, "", err
	}
	aspects, err := buildAspectFilter(q)
	if err != nil {
		return nil, "", err
	}
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}
	v := url.Values{}
	if keywords != "" {
		v.Set("q", keywords)
	}
	if len(q.CategoryIDs) > 0 {
		v.Set("category_ids", strings.Join(q.CategoryIDs, ","))
	}
	if aspects != "" {
		v.Set("aspect_filter", aspects)
	}
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
	return v, market, nil
}

// Refinements asks eBay which sub-categories, conditions and item specifics
// exist for a search, so filters can be written with eBay's exact spelling.
func (c *Client) Refinements(ctx context.Context, q Query) (*Refinement, error) {
	q.Limit = 1
	v, market, err := searchParams(q)
	if err != nil {
		return nil, err
	}
	v.Set("fieldgroups", "CATEGORY_REFINEMENTS,ASPECT_REFINEMENTS,CONDITION_REFINEMENTS")
	var out struct {
		Refinement *Refinement `json:"refinement"`
	}
	if err := c.do(ctx, "/item_summary/search", v, market, q.ShipTo, &out); err != nil {
		return nil, err
	}
	if out.Refinement == nil {
		return &Refinement{}, nil
	}
	return out.Refinement, nil
}
