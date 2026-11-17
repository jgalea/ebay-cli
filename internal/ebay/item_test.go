package ebay

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseItemRef(t *testing.T) {
	cases := []struct {
		in     string
		id     string
		legacy bool
	}{
		{"257787957947", "257787957947", true},
		{"v1|257787957947|0", "v1|257787957947|0", false},
		{"https://www.ebay.de/itm/137818304150?_skw=festplatte+4tb&hash=item20169c6e96:g:7DMAAeSwSCRqn~oI", "137818304150", true},
		{"https://www.ebay.de/itm/some-title-words/137818304150", "137818304150", true},
	}
	for _, c := range cases {
		id, legacy, err := ParseItemRef(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if id != c.id || legacy != c.legacy {
			t.Fatalf("%q: got %q %v want %q %v", c.in, id, legacy, c.id, c.legacy)
		}
	}
	if _, _, err := ParseItemRef("festplatte"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestHTMLToText(t *testing.T) {
	in := `<style>p{color:red}</style><div><b>HGST</b>&nbsp;4&nbsp;TB</div><p>Betriebsstunden:  7819<br>Zustand: gut</p><ul><li>SATA</li><li>3,5&quot;</li></ul><table><tr><td>Reallocated</td><td>0</td></tr></table>`
	got := HTMLToText(in)
	for _, want := range []string{"HGST 4 TB", "Betriebsstunden: 7819\nZustand: gut", "- SATA", `- 3,5"`, "Reallocated | 0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "color") || strings.Contains(got, "<") {
		t.Fatalf("markup left over:\n%s", got)
	}
}

func TestImageURLs(t *testing.T) {
	d := ItemDetail{
		Image:            &Image{ImageURL: "https://i.ebayimg.com/a/s-l1600.jpg"},
		AdditionalImages: []Image{{ImageURL: "https://i.ebayimg.com/b/s-l1600.jpg"}, {ImageURL: "https://i.ebayimg.com/a/s-l1600.jpg"}},
		Description:      `<img src="//example.com/smart.png"><IMG alt=x SRC='https://example.com/c.jpg?x=1&amp;y=2'><img src="data:image/png;base64,AAA">`,
	}
	want := []string{
		"https://i.ebayimg.com/a/s-l1600.jpg",
		"https://i.ebayimg.com/b/s-l1600.jpg",
		"https://example.com/smart.png",
		"https://example.com/c.jpg?x=1&y=2",
	}
	if got := d.ImageURLs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}
