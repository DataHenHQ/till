// Tutorial from from https://github.com/gocolly/colly/blob/master/_examples/basic/basic.go
// modified to integrate with Till
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gocolly/colly/v2"
)

func main() {
	// Instantiate default collector
	c := colly.NewCollector(
		// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
		colly.AllowedDomains("hackerspaces.org", "wiki.hackerspaces.org"),
	)

	// Integration with Till
	proxyUrl, err := url.Parse("http://localhost:2933")
	if err != nil {
		log.Fatal(err)
	}
	tilltransport := http.Transport{
		Proxy:           http.ProxyURL(proxyUrl),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c.WithTransport(&tilltransport)

	// Add custom headers to tell Till what to do
	c.OnRequest(func(req *colly.Request) {
		// Add the header to force a Cache Miss on Till
		req.Headers.Add("X-DH-Cache-Freshness", "now")
	})

	// On every a element which has href attribute call callback
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		fmt.Printf("Link found: %q -> %s\n", e.Text, link)
		// Visit link found on page
		// Only those links are visited which are in AllowedDomains
		c.Visit(e.Request.AbsoluteURL(link))
	})

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// Start scraping on https://hackerspaces.org
	c.Visit("https://hackerspaces.org/")
}
