package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"
)

const BASE_URL = "https://stash.clash.gg/containers/skin-cases?name=&sort=volume&order=desc"

type caseLink struct {
	URL  *url.URL
	Name string
}

type scraper struct {
	client *http.Client
}

func newHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func ScrapeWebsite(ctx context.Context) (map[string]*CaseResult, error) {
	rateLimitedTransport := &RateLimitedTransport{
		limiter:   rate.NewLimiter(rate.Every(500*time.Millisecond), 1),
		transport: newChromeTLSTransport(),
	}

	s := &scraper{
		client: newHTTPClient(rateLimitedTransport),
	}

	caseLinks, err := s.scrapeHomepage(ctx)
	if err != nil {
		return nil, fmt.Errorf("error scraping homepage: %w", err)
	}

	results := make(map[string]*CaseResult, len(caseLinks))

	for index, caseLink := range caseLinks {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := s.scrapeCase(ctx, caseLink)
		if err != nil {
			fmt.Printf("error scraping case %d (%s): %v\n", index, caseLink.Name, err)
			continue
		}

		fmt.Printf("scraped case %d: %s\n", index, caseLink.Name)
		results[caseLink.Name] = result
	}

	return results, nil
}

func (s *scraper) scrapeHomepage(ctx context.Context) ([]caseLink, error) {
	doc, baseURL, _, err := s.fetchDocument(ctx, BASE_URL, true)
	if err != nil {
		return nil, err
	}

	caseLinks := make([]caseLink, 0)
	doc.Find("div.well.result-box.nomargin").Each(func(_ int, selection *goquery.Selection) {
		anchor := selection.ChildrenFiltered("a").First()
		if anchor.Length() == 0 {
			return
		}

		href, ok := anchor.Attr("href")
		if !ok {
			return
		}

		absoluteURL, err := baseURL.Parse(href)
		if err != nil {
			return
		}

		caseName := strings.TrimSpace(anchor.Find("h4").First().Text())
		if caseName == "" {
			caseName = strings.TrimSpace(selection.Find("h4").First().Text())
		}
		if caseName == "" {
			return
		}

		caseLinks = append(caseLinks, caseLink{
			URL:  absoluteURL,
			Name: caseName,
		})
	})

	return caseLinks, nil
}

func getRarity(className string) SkinRarity {
	switch className {
	case "color-classified":
		return SkinRarityPink
	case "color-covert":
		return SkinRarityRed
	case "color-restricted":
		return SkinRarityPurple
	case "color-milspec":
		return SkinRarityBlue
	case "color-rare-item":
		return SkinRarityGold
	default:
		return SkinRarityUnknown
	}
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func (s *scraper) fetchDocument(ctx context.Context, rawURL string, followRedirects bool) (*goquery.Document, *url.URL, int, error) {
	resp, err := s.doRequest(ctx, rawURL, followRedirects)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.Request.URL, resp.StatusCode, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, rawURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, resp.Request.URL, resp.StatusCode, err
	}

	return doc, resp.Request.URL, resp.StatusCode, nil
}

func (s *scraper) doRequest(ctx context.Context, rawURL string, followRedirects bool) (*http.Response, error) {
	client := s.client

	if !followRedirects {
		redirectAware := *s.client
		redirectAware.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
		client = &redirectAware
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func parseListingURLs(doc *goquery.Document, baseURL *url.URL, isKnives bool) ([]string, []string) {
	itemURLs := make([]string, 0)
	specialURLs := make([]string, 0)
	seenItems := make(map[string]struct{})
	seenSpecial := make(map[string]struct{})

	doc.Find("a[href]").Each(func(_ int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if !ok || href == "" {
			return
		}

		absoluteURL, err := baseURL.Parse(href)
		if err != nil {
			return
		}

		switch {
		case !isKnives && isCaseSpecialURL(absoluteURL):
			if _, exists := seenSpecial[absoluteURL.String()]; exists {
				return
			}
			seenSpecial[absoluteURL.String()] = struct{}{}
			specialURLs = append(specialURLs, absoluteURL.String())
		case isKnives && isSpecialItemURL(absoluteURL):
			if _, exists := seenItems[absoluteURL.String()]; exists {
				return
			}
			seenItems[absoluteURL.String()] = struct{}{}
			itemURLs = append(itemURLs, absoluteURL.String())
		case !isKnives && isSkinURL(absoluteURL):
			if _, exists := seenItems[absoluteURL.String()]; exists {
				return
			}
			seenItems[absoluteURL.String()] = struct{}{}
			itemURLs = append(itemURLs, absoluteURL.String())
		}
	})

	return itemURLs, specialURLs
}

func (s *scraper) getKnivesURLs(ctx context.Context, rawURL string) ([]string, error) {
	knifeURLs := make([]string, 0)

	for page := 1; ; page++ {
		pageURL := rawURL + "&page=" + strconv.Itoa(page)
		resp, err := s.doRequest(ctx, pageURL, false)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			resp.Body.Close()
			break
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, pageURL)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		pageSkinURLs, _ := parseListingURLs(doc, resp.Request.URL, true)
		if len(pageSkinURLs) == 0 {
			break
		}

		knifeURLs = append(knifeURLs, pageSkinURLs...)
	}

	return knifeURLs, nil
}

func parsePrice(text string) (float64, error) {
	for _, field := range strings.Fields(strings.TrimSpace(text)) {
		if !strings.HasPrefix(field, "$") {
			continue
		}

		cleaned := strings.ReplaceAll(field, ",", "")
		if len(cleaned) < 2 {
			continue
		}

		return strconv.ParseFloat(cleaned[1:], 64)
	}

	return 0, fmt.Errorf("missing price in %q", text)
}

func parseSkin(doc *goquery.Document, pageURL *url.URL, isKnife bool) (*Skin, error) {
	anchors := doc.Find("div.well.result-box.nomargin h2 a")
	if anchors.Length() < 2 {
		anchors = doc.Find("h1 a")
	}
	if anchors.Length() < 2 {
		return nil, fmt.Errorf("missing skin title anchors")
	}

	skin := &Skin{
		Name:     strings.TrimSpace(anchors.Eq(0).Text()) + " | " + strings.TrimSpace(anchors.Eq(1).Text()),
		Pricing:  make(map[string]*float64),
		Stattrak: doc.Find("div.stattrak").Length() > 0,
		Rarity:   SkinRarityUnknown,
	}

	if isKnife {
		skin.Rarity = SkinRarityGold
	} else {
		classNames, _ := doc.Find("div.quality").First().Attr("class")
		for _, className := range strings.Fields(classNames) {
			if strings.HasPrefix(className, "color") {
				skin.Rarity = getRarity(className)
				break
			}
		}
	}

	doc.Find("div#prices div.btn-group-sm.btn-group-justified").Each(func(_ int, selection *goquery.Selection) {
		classNames, _ := selection.Attr("class")
		if strings.Contains(classNames, "price-bottom-space") {
			return
		}

		spans := selection.Find("a span")
		if spans.Length() == 0 {
			return
		}

		identifierParts := make([]string, 0, spans.Length()-1)
		spans.Each(func(index int, span *goquery.Selection) {
			if index < spans.Length()-1 {
				identifierParts = append(identifierParts, strings.TrimSpace(span.Text()))
			}
		})

		identifier := strings.Join(identifierParts, " ")
		if identifier == "" {
			return
		}

		priceText := strings.ReplaceAll(strings.TrimSpace(spans.Last().Text()), ",", "")
		lowerPriceText := strings.ToLower(priceText)

		switch lowerPriceText {
		case "not possible", "no recent price":
			skin.Pricing[identifier] = nil
		default:
			if len(priceText) < 2 {
				return
			}
			price, err := strconv.ParseFloat(priceText[1:], 64)
			if err != nil {
				return
			}
			skin.Pricing[identifier] = &price
		}
	})

	if minWearValue, ok := doc.Find("div.wear-min-value").First().Attr("data-wearmin"); ok {
		if minWear, err := strconv.ParseFloat(minWearValue, 64); err == nil {
			skin.MinWear = &minWear
		}
	}

	if maxWearValue, ok := doc.Find("div.wear-max-value").First().Attr("data-wearmax"); ok {
		if maxWear, err := strconv.ParseFloat(maxWearValue, 64); err == nil {
			skin.MaxWear = &maxWear
		}
	}

	if src, ok := doc.Find("img.main-skin-img").First().Attr("src"); ok {
		if imageURL, err := pageURL.Parse(src); err == nil {
			image := imageURL.String()
			skin.Img = &image
		}
	}

	return skin, nil
}

func (s *scraper) scrapeSkinPage(ctx context.Context, rawURL string, isKnife bool) (*Skin, error) {
	doc, pageURL, _, err := s.fetchDocument(ctx, rawURL, true)
	if err != nil {
		return nil, err
	}

	return parseSkin(doc, pageURL, isKnife)
}

func (s *scraper) scrapeCase(ctx context.Context, caseLink caseLink) (*CaseResult, error) {
	doc, pageURL, _, err := s.fetchDocument(ctx, caseLink.URL.String(), true)
	if err != nil {
		return nil, err
	}

	result := NewCaseResult()

	doc.Find("a.market-button-item").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		priceText := strings.TrimSpace(selection.Text())
		price, err := parsePrice(priceText)
		if err != nil {
			return true
		}
		result.Price = price
		return false
	})

	skinURLs, specialURLs := parseListingURLs(doc, pageURL, false)
	knifeURLs := make([]string, 0)

	for _, specialURL := range specialURLs {
		pageKnifeURLs, err := s.getKnivesURLs(ctx, specialURL)
		if err != nil {
			return nil, err
		}
		knifeURLs = append(knifeURLs, pageKnifeURLs...)
	}

	for _, skinURL := range skinURLs {
		skin, err := s.scrapeSkinPage(ctx, skinURL, false)
		if err != nil {
			fmt.Printf("error scraping skin %s: %v\n", skinURL, err)
			continue
		}

		result.AddSkin(skin)
	}

	for _, knifeURL := range knifeURLs {
		skin, err := s.scrapeSkinPage(ctx, knifeURL, true)
		if err != nil {
			fmt.Printf("error scraping skin %s: %v\n", knifeURL, err)
			continue
		}

		result.AddSkin(skin)
	}

	return result, nil
}

func isSkinURL(parsedURL *url.URL) bool {
	return strings.HasPrefix(parsedURL.Path, "/skin/")
}

func isSpecialItemURL(parsedURL *url.URL) bool {
	return strings.HasPrefix(parsedURL.Path, "/knife/") || strings.HasPrefix(parsedURL.Path, "/glove/")
}

func isCaseSpecialURL(parsedURL *url.URL) bool {
	query := parsedURL.Query()
	return query.Get("Knives") == "1" || query.Get("Gloves") == "1"
}
