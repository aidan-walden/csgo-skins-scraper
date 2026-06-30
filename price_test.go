package main

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// Verifies the case-price selector against a real saved case page.
// Set CASE_HTML to the saved Prisma Case HTML to run.
func TestCasePriceSelector(t *testing.T) {
	path := os.Getenv("CASE_HTML")
	if path == "" {
		t.Skip("set CASE_HTML to a saved case page")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	var price float64
	doc.Find("div.market-item").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if strings.TrimSpace(sel.Find("span.label").First().Text()) != "Steam price" {
			return true
		}
		p, err := parsePrice(sel.Find("span.value").First().Text())
		if err != nil {
			return true
		}
		price = p
		return false
	})

	if price != 1.95 {
		t.Fatalf("expected 1.95, got %v", price)
	}
}

// Verifies knife discovery on the ?Knives=1 page now that knives live under
// /skin/. Set KNIVES_HTML to the saved Prisma Case Knives page to run.
func TestKnivesListingDiscovery(t *testing.T) {
	path := os.Getenv("KNIVES_HTML")
	if path == "" {
		t.Skip("set KNIVES_HTML to a saved ?Knives=1 page")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	base, _ := url.Parse("https://stash.clash.gg/case/274/Prisma-Case?Knives=1")
	items, _ := parseListingURLs(doc, base, true)
	if len(items) != 24 {
		t.Fatalf("expected 24 knife URLs, got %d", len(items))
	}
}
