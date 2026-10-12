package ebay

import "testing"

func TestBuildFilterPriceNeedsCurrency(t *testing.T) {
	if _, err := buildFilter(Query{MaxPrice: "400"}); err == nil {
		t.Fatal("expected an error when a price filter has no currency")
	}
}

func TestBuildFilter(t *testing.T) {
	got, err := buildFilter(Query{
		MinPrice:        "100",
		MaxPrice:        "400",
		Currency:        "EUR",
		Conditions:      []string{"USED"},
		SellerLoc:       "de",
		IncludeAuctions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "price:[100..400],priceCurrency:EUR,conditions:{USED},itemLocationCountry:DE,buyingOptions:{AUCTION|FIXED_PRICE}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildFilterEmpty(t *testing.T) {
	got, err := buildFilter(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected no filter, got %q", got)
	}
}

func TestResolveMarket(t *testing.T) {
	for in, want := range map[string]string{
		"de":      "EBAY_DE",
		"DE":      "EBAY_DE",
		"uk":      "EBAY_GB",
		"EBAY_ES": "EBAY_ES",
		"ebay_es": "EBAY_ES",
	} {
		got, err := ResolveMarket(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
	if _, err := ResolveMarket("atlantis"); err == nil {
		t.Fatal("expected an error for an unknown marketplace")
	}
}

func TestResolveSort(t *testing.T) {
	got, err := resolveSort("new")
	if err != nil || got != "newlyListed" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := resolveSort("sideways"); err == nil {
		t.Fatal("expected an error for an unknown sort")
	}
}

func TestActivePrefersStandingBid(t *testing.T) {
	it := Item{
		Price:           &Money{Value: "1.00", Currency: "EUR"},
		CurrentBidPrice: &Money{Value: "200.00", Currency: "EUR"},
		BidCount:        44,
	}
	if got := it.Active(); got.Value != "200.00" {
		t.Fatalf("got %q, want the standing bid", got.Value)
	}

	noBids := Item{Price: &Money{Value: "499.00", Currency: "EUR"}}
	if got := noBids.Active(); got.Value != "499.00" {
		t.Fatalf("got %q, want the listed price", got.Value)
	}
}

func TestShippingPicksCheapestAndReportsAbsence(t *testing.T) {
	it := Item{ShippingOptions: []ShippingOption{
		{ShippingCost: &Money{Value: "17.49", Currency: "EUR"}},
		{ShippingCost: &Money{Value: "7.69", Currency: "EUR"}},
	}}
	got, ok := it.Shipping()
	if !ok || got.Value != "7.69" {
		t.Fatalf("got %q (%v), want 7.69", got.Value, ok)
	}

	if _, ok := (Item{}).Shipping(); ok {
		t.Fatal("expected no shipping quote when eBay omits the container")
	}
}

func TestActiveOpeningBidWithoutBids(t *testing.T) {
	it := Item{
		CurrentBidPrice: &Money{Value: "10.00", Currency: "EUR"},
		BuyingOptions:   []string{"AUCTION"},
	}
	if got := it.Active(); got == nil || got.Value != "10.00" {
		t.Fatalf("got %v, want the opening bid", got)
	}
}

func TestDefaultShipTo(t *testing.T) {
	for market, want := range map[string]string{"EBAY_DE": "DE", "EBAY_ES": "ES", "EBAY_GB": "GB", "ebay_at": "AT", "EBAY_MOTORS_US": ""} {
		if got := DefaultShipTo(market); got != want {
			t.Fatalf("%s: got %q want %q", market, got, want)
		}
	}
}

func TestParseAspect(t *testing.T) {
	a, err := ParseAspect("Speicherkapazität=2TB| 3 TB |4 TB")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Speicherkapazität" || len(a.Values) != 3 || a.Values[1] != "3 TB" {
		t.Fatalf("got %+v", a)
	}
	for _, bad := range []string{"Speicherkapazität", "=4 TB", "Speicherkapazität=", "X=|"} {
		if _, err := ParseAspect(bad); err == nil {
			t.Fatalf("%q: expected an error", bad)
		}
	}
}

func TestSearchParamsCategoryAspectsConditions(t *testing.T) {
	v, market, err := searchParams(Query{
		Market:          "de",
		CategoryIDs:     []string{"56083"},
		Aspects:         []Aspect{{Name: "Speicherkapazität", Values: []string{"2TB", "4 TB"}}, {Name: "Schnittstelle", Values: []string{"SATA III"}}},
		ConditionIDs:    []string{"1000", "3000"},
		IncludeAuctions: true,
		Sort:            "ending",
	})
	if err != nil {
		t.Fatal(err)
	}
	if market != "EBAY_DE" {
		t.Fatalf("market %q", market)
	}
	if v.Has("q") {
		t.Fatal("no keywords given, q must be absent")
	}
	if got := v.Get("category_ids"); got != "56083" {
		t.Fatalf("category_ids %q", got)
	}
	if got, want := v.Get("aspect_filter"), "categoryId:56083,Speicherkapazität:{2TB|4 TB},Schnittstelle:{SATA III}"; got != want {
		t.Fatalf("aspect_filter got %q want %q", got, want)
	}
	if got, want := v.Get("filter"), "conditionIds:{1000|3000},buyingOptions:{AUCTION|FIXED_PRICE}"; got != want {
		t.Fatalf("filter got %q want %q", got, want)
	}
	if v.Get("sort") != "endingSoonest" {
		t.Fatalf("sort %q", v.Get("sort"))
	}
}

func TestSearchParamsNeedsKeywordsOrCategory(t *testing.T) {
	if _, _, err := searchParams(Query{Market: "de"}); err == nil {
		t.Fatal("expected an error without keywords and category")
	}
	if _, _, err := searchParams(Query{Market: "de", Keywords: "x", Aspects: []Aspect{{Name: "A", Values: []string{"b"}}}}); err == nil {
		t.Fatal("expected an error for aspects without a category")
	}
}

func TestNotDefectiveExcludes7000(t *testing.T) {
	for _, id := range NotDefectiveConditionIDs {
		if id == "7000" {
			t.Fatal("7000 is for parts or not working")
		}
	}
}
