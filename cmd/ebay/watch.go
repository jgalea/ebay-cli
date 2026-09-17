package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jgalea/ebay-cli/internal/ebay"
)

type watch struct {
	Name     string   `json:"name"`
	Keywords string   `json:"keywords"`
	Market   string   `json:"market"`
	ShipTo   string   `json:"shipTo"`
	Min      string   `json:"min,omitempty"`
	Max      string   `json:"max,omitempty"`
	Currency string   `json:"currency"`
	Auctions bool     `json:"auctions"`
	Seen     []string `json:"seen"`
}

type watchFile struct {
	Watches []watch `json:"watches"`
}

func watchPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "ebay-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "watches.json"), nil
}

func loadWatches() (*watchFile, string, error) {
	path, err := watchPath()
	if err != nil {
		return nil, "", err
	}
	var wf watchFile
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &wf, path, nil
	}
	if err != nil {
		return nil, "", err
	}
	if err := json.Unmarshal(b, &wf); err != nil {
		return nil, "", fmt.Errorf("%s is not readable as JSON: %w", path, err)
	}
	return &wf, path, nil
}

func saveWatches(wf *watchFile, path string) error {
	b, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func cmdWatch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("watch needs a subcommand: add, list, remove, run")
	}
	switch args[0] {
	case "add":
		return watchAdd(args[1:])
	case "list":
		return watchList()
	case "remove", "rm":
		return watchRemove(args[1:])
	case "run":
		return watchRun(args[1:])
	default:
		return fmt.Errorf("unknown watch subcommand %q", args[0])
	}
}

func watchAdd(args []string) error {
	fs := flag.NewFlagSet("watch add", flag.ContinueOnError)
	var f searchFlags
	f.bind(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("usage: ebay watch add <name> <keywords>")
	}
	name := rest[0]
	keywords := strings.Join(rest[1:], " ")

	wf, path, err := loadWatches()
	if err != nil {
		return err
	}
	for _, w := range wf.Watches {
		if w.Name == name {
			return fmt.Errorf("a watch named %q already exists", name)
		}
	}
	wf.Watches = append(wf.Watches, watch{
		Name:     name,
		Keywords: keywords,
		Market:   f.market,
		ShipTo:   f.shipTo,
		Min:      f.min,
		Max:      f.max,
		Currency: f.currency,
		Auctions: f.auctions,
	})
	if err := saveWatches(wf, path); err != nil {
		return err
	}
	fmt.Printf("watching %q on %s: %s\n", name, strings.ToUpper(f.market), keywords)
	fmt.Println("first `ebay watch run` records what is already listed, so only later arrivals are reported")
	return nil
}

func watchList() error {
	wf, _, err := loadWatches()
	if err != nil {
		return err
	}
	if len(wf.Watches) == 0 {
		fmt.Println("no watches yet")
		return nil
	}
	for _, w := range wf.Watches {
		price := ""
		if w.Min != "" || w.Max != "" {
			price = fmt.Sprintf("  %s-%s %s", w.Min, w.Max, w.Currency)
		}
		fmt.Printf("%-16s %s [%s]%s  %d seen\n", w.Name, w.Keywords, strings.ToUpper(w.Market), price, len(w.Seen))
	}
	return nil
}

func watchRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ebay watch remove <name>")
	}
	wf, path, err := loadWatches()
	if err != nil {
		return err
	}
	out := wf.Watches[:0]
	found := false
	for _, w := range wf.Watches {
		if w.Name == args[0] {
			found = true
			continue
		}
		out = append(out, w)
	}
	if !found {
		return fmt.Errorf("no watch named %q", args[0])
	}
	wf.Watches = out
	if err := saveWatches(wf, path); err != nil {
		return err
	}
	fmt.Printf("removed %q\n", args[0])
	return nil
}

func watchRun(args []string) error {
	only := ""
	if len(args) == 1 {
		only = args[0]
	}
	wf, path, err := loadWatches()
	if err != nil {
		return err
	}
	if len(wf.Watches) == 0 {
		return fmt.Errorf("no watches configured")
	}
	creds, err := ebay.LoadCredentials()
	if err != nil {
		return err
	}
	client := ebay.New(creds)

	ran := 0
	for i := range wf.Watches {
		w := &wf.Watches[i]
		if only != "" && w.Name != only {
			continue
		}
		ran++

		res, err := client.Search(context.Background(), ebay.Query{
			Keywords:        w.Keywords,
			Market:          w.Market,
			ShipTo:          w.ShipTo,
			MinPrice:        w.Min,
			MaxPrice:        w.Max,
			Currency:        w.Currency,
			IncludeAuctions: w.Auctions,
			Sort:            "new",
			Limit:           100,
		})
		if err != nil {
			return fmt.Errorf("watch %q: %w", w.Name, err)
		}

		seen := make(map[string]bool, len(w.Seen))
		for _, id := range w.Seen {
			seen[id] = true
		}
		first := len(w.Seen) == 0

		var fresh []ebay.Item
		for _, it := range res.ItemSummaries {
			if !seen[it.ItemID] {
				seen[it.ItemID] = true
				fresh = append(fresh, it)
			}
		}

		ids := make([]string, 0, len(seen))
		for id := range seen {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		w.Seen = ids

		if first {
			fmt.Printf("%s: recorded %d existing listings, will report new ones from now on\n", w.Name, len(fresh))
			continue
		}
		if len(fresh) == 0 {
			continue
		}
		fmt.Printf("%s: %d new\n\n", w.Name, len(fresh))
		printResults(os.Stdout, &ebay.SearchResult{
			Total:         len(fresh),
			ItemSummaries: fresh,
		}, ebay.Query{Keywords: w.Keywords, Market: w.Market, IncludeAuctions: w.Auctions})
	}

	if ran == 0 {
		return fmt.Errorf("no watch named %q", only)
	}
	return saveWatches(wf, path)
}
