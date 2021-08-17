package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {

	// setup the proxy connection to Till
	proxyUrl, err := url.Parse("http://localhost:2933")
	myClient := &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(proxyUrl),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	//
	// Example 1: GET request
	//

	// create a new GET request
	greq, err := http.NewRequest("GET", "https://fetchtest.datahen.com/echo/request", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Add the header to force a Cache Miss on Till
	greq.Header.Add("X-DH-Cache-Freshness", "now")

	// Do the actual request
	gresp, err := myClient.Do(greq)
	if err != nil {
		log.Fatal(err)
	}

	// print out the response
	grout, err := httputil.DumpResponse(gresp, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("-------------------")
	fmt.Println("GET RESPONSE:")
	fmt.Println("-------------------")
	fmt.Println(string(grout))

	//
	// Example 2: Post request
	//

	// create a new POST request
	jsonData := `{"hello":"world"}`

	preq, err := http.NewRequest("POST", "https://postman-echo.com/post", bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		log.Fatal(err)
	}

	// Add the header to force a Cache Miss on Till
	preq.Header.Add("X-DH-Cache-Freshness", "now")

	preq.Header.Set("Content-Type", "application/json")
	preq.Header.Set("Accept", "*/*")

	// Do the actual request
	presp, err := myClient.Do(preq)
	if err != nil {
		log.Fatal(err)
	}

	// print out the response
	prout, err := httputil.DumpResponse(presp, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("-------------------")
	fmt.Println("POST RESPONSE:")
	fmt.Println("-------------------")
	fmt.Println(string(prout))

}
