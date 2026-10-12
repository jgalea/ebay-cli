package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestParseInterspersed(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	var f searchFlags
	f.bind(fs)
	pos, err := parseInterspersed(fs, []string{"flair", "-m", "es", "58", "--auctions", "--sort", "-price", "--", "--literal"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"flair", "58", "--literal"}; !reflect.DeepEqual(pos, want) {
		t.Fatalf("got %v want %v", pos, want)
	}
	if f.market != "es" || f.shipTo != "" || !f.auctions || f.sort != "-price" {
		t.Fatalf("flags not parsed: %+v", f)
	}
}
