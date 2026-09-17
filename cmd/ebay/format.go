package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jgalea/ebay-cli/internal/ebay"
)

func money(m *ebay.Money) string {
	if m == nil {
		return "?"
	}
	return m.Value + " " + m.Currency
}

func printResults(w io.Writer, res *ebay.SearchResult, q ebay.Query) {
	for _, warn := range res.Warnings {
		fmt.Fprintln(w, "note:", warn.Message)
	}
	if len(res.ItemSummaries) == 0 {
		fmt.Fprintf(w, "no listings for %q on %s\n", q.Keywords, strings.ToUpper(q.Market))
		if !q.IncludeAuctions && !q.FixedOnly {
			fmt.Fprintln(w, "eBay hides auction-only listings by default; try --auctions")
		}
		return
	}

	for _, it := range res.ItemSummaries {
		fmt.Fprintln(w, it.Title)

		line := "  " + money(it.Active())
		if ship, ok := it.Shipping(); ok {
			if ship.Value == "0.0" || ship.Value == "0.00" {
				line += " + free shipping"
			} else {
				line += " + " + money(&ship) + " shipping"
			}
		} else {
			line += " + shipping not quoted"
		}
		if it.IsAuction() {
			line += fmt.Sprintf("  [auction, %d bids", it.BidCount)
			if it.ItemEndDate != nil {
				line += ", ends " + humanUntil(*it.ItemEndDate)
			}
			line += "]"
		}
		fmt.Fprintln(w, line)

		var meta []string
		if it.Condition != "" {
			meta = append(meta, it.Condition)
		}
		if it.ItemLocation != nil && it.ItemLocation.Country != "" {
			where := it.ItemLocation.Country
			if it.ItemLocation.City != "" {
				where = it.ItemLocation.City + ", " + where
			}
			meta = append(meta, where)
		}
		if it.Seller != nil && it.Seller.Username != "" {
			s := it.Seller.Username
			if it.Seller.FeedbackPercentage != "" {
				s += fmt.Sprintf(" (%s%%, %d)", it.Seller.FeedbackPercentage, it.Seller.FeedbackScore)
			}
			meta = append(meta, s)
		}
		if len(meta) > 0 {
			fmt.Fprintln(w, "  "+strings.Join(meta, " · "))
		}
		fmt.Fprintln(w, "  "+it.ItemWebURL)
		fmt.Fprintln(w)
	}

	shown := len(res.ItemSummaries)
	fmt.Fprintf(w, "%d shown of %d total on %s\n", shown, res.Total, strings.ToUpper(q.Market))
}

func humanUntil(t time.Time) string {
	d := time.Until(t)
	if d < 0 {
		return "ended"
	}
	if d < time.Hour {
		return fmt.Sprintf("in %dm", int(d.Minutes()))
	}
	if d < 48*time.Hour {
		return fmt.Sprintf("in %dh", int(d.Hours()))
	}
	return fmt.Sprintf("in %dd", int(d.Hours()/24))
}
