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
	pos, err := parseInterspersed(fs, []string{"festplatte", "-m", "de", "4tb", "--auctions", "--sort", "-price", "--aspect", "Schnittstelle=SATA III", "--", "--literal"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"festplatte", "4tb", "--literal"}; !reflect.DeepEqual(pos, want) {
		t.Fatalf("got %v want %v", pos, want)
	}
	if !f.auctions || f.sort != "-price" || len(f.aspects) != 1 || f.shipTo != "" {
		t.Fatalf("flags not parsed: %+v", f)
	}
}

func TestNoDefectQuery(t *testing.T) {
	f := searchFlags{noDefect: true, currency: "EUR"}
	q, err := f.query("x")
	if err != nil {
		t.Fatal(err)
	}
	if len(q.ConditionIDs) == 0 {
		t.Fatal("expected condition ids")
	}
	f.conditionIDs = "3000"
	if _, err := f.query("x"); err == nil {
		t.Fatal("expected a contradiction error")
	}
}
