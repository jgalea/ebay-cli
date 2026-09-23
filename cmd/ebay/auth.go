package main

import (
	"context"
	"flag"
	"fmt"
	"runtime"

	"github.com/jgalea/ebay-cli/internal/ebay"
)

func cmdAuth(args []string) error {
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	check := fs.Bool("check", false, "verify the stored credentials work")
	if err := fs.Parse(args); err != nil {
		return err
	}

	creds, err := ebay.LoadCredentials()
	if err != nil {
		printSetup()
		return err
	}

	if !*check {
		fmt.Printf("client id found, ending %s\n", tail(creds.ClientID))
		fmt.Println("run `ebay auth --check` to confirm it mints a token")
		return nil
	}

	if _, err := ebay.New(creds).Search(context.Background(), ebay.Query{
		Keywords: "espresso",
		Market:   "de",
		Limit:    1,
	}); err != nil {
		return fmt.Errorf("credentials did not work: %w", err)
	}
	fmt.Println("credentials work")
	return nil
}

func tail(s string) string {
	if len(s) <= 6 {
		return s
	}
	return "..." + s[len(s)-6:]
}

func printSetup() {
	fmt.Print(`
Set up once:

  1. Create an eBay developer account and a Production keyset at
     https://developer.ebay.com/my/keys
     The App ID is the client id and the Cert ID is the client secret.
`)
	if runtime.GOOS == "darwin" {
		fmt.Print(`  2. Store the two values in the Keychain, entering each when prompted so the
     value never appears in shell history:

     security add-generic-password -a "$USER" -s claude-ebay-client-id -U -w
     security add-generic-password -a "$USER" -s claude-ebay-client-secret -U -w

`)
	}
	path, _ := ebay.CredentialsPath()
	step, perms := "2. Save", ""
	if runtime.GOOS == "darwin" {
		step = "3. Or save"
	}
	if runtime.GOOS != "windows" {
		perms = "\n     then chmod 600 it so only you can read it"
	}
	fmt.Printf(`  %s them in %s%s:

     {"client_id": "...", "client_secret": "..."}

EBAY_CLIENT_ID and EBAY_CLIENT_SECRET work too, and are read first.

`, step, path, perms)
}
