package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) < 1 {
		usage()
		return 2
	}
	var err error
	switch args[0] {
	case "search":
		err = cmdSearch(args[1:])
	case "watch":
		err = cmdWatch(args[1:])
	case "auth":
		err = cmdAuth(args[1:])
	case "markets":
		err = cmdMarkets(args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Print(`ebay - search eBay marketplaces from the terminal

Usage:
  ebay search <keywords> [flags]
  ebay watch <add|list|remove|run> [args]
  ebay auth [--check]
  ebay markets

Search flags:
  -m, --market     marketplace: de, es, it, fr, gb, nl, us ... (default de)
      --ship-to    destination country for shipping quotes (default PT)
      --min        minimum price
      --max        maximum price
      --currency   currency for price filters (default EUR)
      --condition  new, used, or both (default both)
      --seller-in  only sellers located in this country
      --auctions   include auction listings (eBay hides them by default)
      --fixed      buy-it-now listings only
      --sort       price, -price, new, ending
      --limit      results per page, 1-200 (default 50)
      --json       raw JSON output

Watch:
  ebay watch add <name> <keywords> [search flags]
  ebay watch run [name]     print only listings not seen before
  ebay watch list
  ebay watch remove <name>

Credentials come from EBAY_CLIENT_ID and EBAY_CLIENT_SECRET, the macOS
Keychain, or a credentials file. Run "ebay auth" for setup instructions.
`)
}
