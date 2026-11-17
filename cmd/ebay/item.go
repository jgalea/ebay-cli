package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jgalea/ebay-cli/internal/ebay"
)

type itemOutput struct {
	*ebay.ItemDetail
	DescriptionText string   `json:"descriptionText"`
	Images          []string `json:"images"`
	ImageFiles      []string `json:"imageFiles,omitempty"`
}

func cmdItem(args []string) error {
	fs := flag.NewFlagSet("item", flag.ContinueOnError)
	var (
		market, shipTo, imgDir string
		asJSON, rawHTML        bool
	)
	fs.StringVar(&market, "market", "de", "marketplace")
	fs.StringVar(&market, "m", "de", "marketplace")
	fs.StringVar(&shipTo, "ship-to", "", "destination country for shipping quotes (default: the marketplace country)")
	fs.StringVar(&imgDir, "images", "", "download all pictures into this directory")
	fs.BoolVar(&asJSON, "json", false, "JSON output")
	fs.BoolVar(&rawHTML, "html", false, "print the description as HTML instead of text")
	refs, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return fmt.Errorf("usage: ebay item <item number|v1|…|…|listing URL>... [flags]")
	}
	creds, err := ebay.LoadCredentials()
	if err != nil {
		return err
	}
	client := ebay.New(creds)
	ctx := context.Background()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	for i, ref := range refs {
		d, err := client.GetItem(ctx, ref, market, shipTo)
		if err != nil {
			return fmt.Errorf("%s: %w", ref, err)
		}
		out := itemOutput{
			ItemDetail:      d,
			DescriptionText: ebay.HTMLToText(d.Description),
			Images:          d.ImageURLs(),
		}
		if imgDir != "" {
			files, err := downloadImages(client.HTTP, imgDir, itemNumber(d), out.Images)
			if err != nil {
				return fmt.Errorf("%s: %w", ref, err)
			}
			out.ImageFiles = files
		}
		if asJSON {
			if err := enc.Encode(out); err != nil {
				return err
			}
			continue
		}
		if i > 0 {
			fmt.Println(strings.Repeat("─", 72))
		}
		printItem(os.Stdout, out, rawHTML)
	}
	return nil
}

func itemNumber(d *ebay.ItemDetail) string {
	if d.LegacyItemID != "" {
		return d.LegacyItemID
	}
	return strings.ReplaceAll(d.ItemID, "|", "_")
}

func printItem(w io.Writer, o itemOutput, rawHTML bool) {
	d := o.ItemDetail
	fmt.Fprintln(w, d.Title)
	if d.Subtitle != "" {
		fmt.Fprintln(w, "  "+d.Subtitle)
	}
	it := d.Summary()
	line := "  " + money(it.Active())
	if ship, ok := it.Shipping(); ok {
		line += " + " + money(&ship) + " shipping"
	} else {
		line += " + shipping not quoted"
	}
	if it.IsAuction() {
		line += fmt.Sprintf("  [auction, %d bids", it.BidCount)
		if it.ItemEndDate != nil {
			line += ", ends " + humanUntil(*it.ItemEndDate) + " (" + it.ItemEndDate.Local().Format("02.01.2006 15:04") + ")"
		}
		line += "]"
	}
	fmt.Fprintln(w, line)
	fmt.Fprintf(w, "  condition: %s (%s)\n", d.Condition, d.ConditionID)
	if d.ConditionDescription != "" {
		fmt.Fprintln(w, "  condition notes: "+d.ConditionDescription)
	}
	if d.Seller != nil {
		fmt.Fprintf(w, "  seller: %s (%s%%, %d)", d.Seller.Username, d.Seller.FeedbackPercentage, d.Seller.FeedbackScore)
		if d.ItemLocation != nil {
			fmt.Fprintf(w, " · %s %s", d.ItemLocation.Country, d.ItemLocation.City)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "  id: %s · %s\n", itemNumber(d), d.ItemWebURL)
	if len(d.LocalizedAspects) > 0 {
		fmt.Fprintln(w, "\nItem specifics:")
		for _, a := range d.LocalizedAspects {
			fmt.Fprintf(w, "  %s: %s\n", a.Name, a.Value)
		}
	}
	fmt.Fprintln(w, "\nDescription:")
	desc := o.DescriptionText
	if rawHTML {
		desc = d.Description
	}
	if strings.TrimSpace(desc) == "" {
		desc = d.ShortDescription
	}
	if strings.TrimSpace(desc) == "" {
		desc = "(none)"
	}
	for _, l := range strings.Split(desc, "\n") {
		fmt.Fprintln(w, "  "+l)
	}
	pics := o.Images
	if len(o.ImageFiles) > 0 {
		pics = o.ImageFiles
	}
	if len(pics) > 0 {
		fmt.Fprintf(w, "\nPictures (%d):\n", len(pics))
		for _, p := range pics {
			fmt.Fprintln(w, "  "+p)
		}
	}
	fmt.Fprintln(w)
}

func downloadImages(hc *http.Client, dir, prefix string, urls []string) ([]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	var files []string
	for i, u := range urls {
		ext := ".jpg"
		if pu, err := url.Parse(u); err == nil {
			if e := strings.ToLower(path.Ext(pu.Path)); e == ".jpg" || e == ".jpeg" || e == ".png" || e == ".webp" || e == ".gif" {
				ext = e
			}
		}
		name := filepath.Join(dir, fmt.Sprintf("%s-%02d%s", prefix, i+1, ext))
		if err := fetchFile(hc, u, name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: picture %s: %v\n", u, err)
			continue
		}
		files = append(files, name)
	}
	return files, nil
}

func fetchFile(hc *http.Client, u, name string) error {
	resp, err := hc.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
