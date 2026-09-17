package ebay

import "time"

type Money struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Location struct {
	Country    string `json:"country"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
}

type ShippingOption struct {
	ShippingCostType string `json:"shippingCostType"`
	ShippingCost     *Money `json:"shippingCost"`
}

type Seller struct {
	Username           string `json:"username"`
	FeedbackScore      int    `json:"feedbackScore"`
	FeedbackPercentage string `json:"feedbackPercentage"`
}

type Item struct {
	ItemID           string           `json:"itemId"`
	Title            string           `json:"title"`
	Condition        string           `json:"condition"`
	Price            *Money           `json:"price"`
	CurrentBidPrice  *Money           `json:"currentBidPrice"`
	BidCount         int              `json:"bidCount"`
	BuyingOptions    []string         `json:"buyingOptions"`
	ItemWebURL       string           `json:"itemWebUrl"`
	ItemLocation     *Location        `json:"itemLocation"`
	Seller           *Seller          `json:"seller"`
	ShippingOptions  []ShippingOption `json:"shippingOptions"`
	ItemEndDate      *time.Time       `json:"itemEndDate"`
	ItemCreationDate *time.Time       `json:"itemCreationDate"`
}

type SearchResult struct {
	Total         int    `json:"total"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	ItemSummaries []Item `json:"itemSummaries"`
	Warnings      []struct {
		Message string `json:"message"`
	} `json:"warnings"`
}

// Shipping returns the cheapest quoted shipping cost, and whether one was quoted
// at all. eBay omits the container entirely for some listings.
func (i Item) Shipping() (Money, bool) {
	var best *Money
	for _, o := range i.ShippingOptions {
		if o.ShippingCost == nil {
			continue
		}
		if best == nil || lessMoney(*o.ShippingCost, *best) {
			c := *o.ShippingCost
			best = &c
		}
	}
	if best == nil {
		return Money{}, false
	}
	return *best, true
}

// Active returns the price a buyer would act on right now: the standing bid for
// an auction that has bids, otherwise the listed price.
func (i Item) Active() *Money {
	if i.CurrentBidPrice != nil && i.BidCount > 0 {
		return i.CurrentBidPrice
	}
	return i.Price
}

func (i Item) IsAuction() bool {
	for _, b := range i.BuyingOptions {
		if b == "AUCTION" {
			return true
		}
	}
	return false
}
