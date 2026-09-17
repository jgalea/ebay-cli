package main

import (
	"fmt"
	"sort"

	"github.com/jgalea/ebay-cli/internal/ebay"
)

func cmdMarkets([]string) error {
	codes := make([]string, 0, len(ebay.Markets))
	for c := range ebay.Markets {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, c := range codes {
		fmt.Printf("%-4s %s\n", c, ebay.Markets[c])
	}
	return nil
}
