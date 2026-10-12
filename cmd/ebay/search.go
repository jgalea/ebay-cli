package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jgalea/ebay-cli/internal/ebay"
)

type searchFlags struct {
	market    string
	shipTo    string
	min       string
	max       string
	currency  string
	condition string
	sellerIn  string
	auctions  bool
	fixed     bool
	sort      string
	limit     int
	asJSON    bool
}

func (f *searchFlags) bind(fs *flag.FlagSet) {
	fs.StringVar(&f.market, "market", "de", "marketplace")
	fs.StringVar(&f.market, "m", "de", "marketplace")
	fs.StringVar(&f.shipTo, "ship-to", "", "destination country for shipping quotes (default: the marketplace country)")
	fs.StringVar(&f.min, "min", "", "minimum price")
	fs.StringVar(&f.max, "max", "", "maximum price")
	fs.StringVar(&f.currency, "currency", "EUR", "currency for price filters")
	fs.StringVar(&f.condition, "condition", "", "new, used, or both")
	fs.StringVar(&f.sellerIn, "seller-in", "", "only sellers in this country")
	fs.BoolVar(&f.auctions, "auctions", false, "include auction listings")
	fs.BoolVar(&f.fixed, "fixed", false, "buy-it-now only")
	fs.StringVar(&f.sort, "sort", "", "price, -price, new, ending")
	fs.IntVar(&f.limit, "limit", 50, "results per page")
	fs.BoolVar(&f.asJSON, "json", false, "raw JSON output")
}

func (f *searchFlags) query(keywords string) (ebay.Query, error) {
	q := ebay.Query{
		Keywords:        keywords,
		Market:          f.market,
		ShipTo:          f.shipTo,
		Limit:           f.limit,
		Sort:            f.sort,
		MinPrice:        f.min,
		MaxPrice:        f.max,
		Currency:        f.currency,
		SellerLoc:       f.sellerIn,
		IncludeAuctions: f.auctions,
		FixedOnly:       f.fixed,
	}
	switch strings.ToLower(f.condition) {
	case "", "both", "any":
	case "new":
		q.Conditions = []string{"NEW"}
	case "used":
		q.Conditions = []string{"USED"}
	default:
		return q, fmt.Errorf("unknown condition %q (new, used, both)", f.condition)
	}
	if f.auctions && f.fixed {
		return q, fmt.Errorf("--auctions and --fixed contradict each other")
	}
	return q, nil
}

// parseInterspersed lets flags appear before, between or after positional
// arguments, which the standard flag package does not allow on its own. A
// literal "--" ends flag parsing.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(pos, rest...), nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

func cmdSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	var f searchFlags
	f.bind(fs)
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	keywords := strings.Join(pos, " ")
	if keywords == "" {
		return fmt.Errorf("nothing to search for")
	}

	q, err := f.query(keywords)
	if err != nil {
		return err
	}

	creds, err := ebay.LoadCredentials()
	if err != nil {
		return err
	}
	res, err := ebay.New(creds).Search(context.Background(), q)
	if err != nil {
		return err
	}

	if f.asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}

	printResults(os.Stdout, res, q)
	return nil
}
