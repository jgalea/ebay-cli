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
