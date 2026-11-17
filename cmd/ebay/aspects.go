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

// cmdAspects shows which sub-categories, conditions and item specifics eBay
// knows for a search, with eBay's exact spelling for --category, --aspect and
// --condition-ids.
func cmdAspects(args []string) error {
	fs := flag.NewFlagSet("aspects", flag.ContinueOnError)
	var f searchFlags
	f.bind(fs)
	top := fs.Int("top", 30, "values shown per item specific")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	keywords := strings.Join(pos, " ")
	if keywords == "" && f.category == "" {
		return fmt.Errorf("usage: ebay aspects [keywords] [--category id] [search flags]")
	}
	q, err := f.query(keywords)
	if err != nil {
		return err
	}
	creds, err := ebay.LoadCredentials()
	if err != nil {
		return err
	}
	r, err := ebay.New(creds).Refinements(context.Background(), q)
	if err != nil {
		return err
	}
	if f.asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	if len(r.CategoryDistributions) > 0 {
		fmt.Println("Categories (--category):")
		for _, c := range r.CategoryDistributions {
			fmt.Printf("  %-8s %s (%d)\n", c.CategoryID, c.CategoryName, c.MatchCount)
		}
		fmt.Println()
	}
	if len(r.ConditionDistributions) > 0 {
		fmt.Println("Conditions (--condition-ids):")
		for _, c := range r.ConditionDistributions {
			fmt.Printf("  %-8s %s (%d)\n", c.ConditionID, c.Condition, c.MatchCount)
		}
		fmt.Println()
	}
	if len(r.AspectDistributions) == 0 {
		if len(q.CategoryIDs) == 0 {
			fmt.Println("eBay returns item specifics only for a single leaf category; add --category")
		}
		return nil
	}
	fmt.Println(`Item specifics (--aspect "Name=value1|value2"):`)
	for _, a := range r.AspectDistributions {
		fmt.Printf("  %s\n", a.Name)
		for i, v := range a.Values {
			if *top > 0 && i >= *top {
				fmt.Printf("      … %d more\n", len(a.Values)-i)
				break
			}
			fmt.Printf("      %s (%d)\n", v.Value, v.MatchCount)
		}
	}
	return nil
}
