# DataHen Till
[![Gitter](https://img.shields.io/gitter/room/DataHenHQ/till?logo=gitter&style=for-the-badge)](https://gitter.im/DataHenHQ/till)

![](img/icons8-spade.svg)

**DataHen Till** is a standalone tool that runs alongside your web scraper, and instantly makes your existing web scraper unblockable, scalable, and maintainable, without requiring any code changes on your scraper code. 

Based on [DataHen](https://www.datahen.com)'s experience in almost a decade of scraping billions of pages from thousands of some of the top largest websites in the world, we realized that there's got to be a better way so that the typical developers can easily build web scrapers in their preferred programming language, and be able to scale their scrapers easily.

Till is designed primarily to increase developer happiness, and to follow best practices that DataHen has accumulated over the years of scraping at a massive scale.

## Problems with Web Scraping


Web scraping is easy to get started, but proved to be very difficult to master at scale. Scraping 10,000 records, can easily be done with a simple web scraper scripts in any language, but imagine trying to scrape millions or even billions of pages. You would need to to architect and build features that allows you to unblock, scale and maintain your scrapers. 


The following problems related to scaling are what DataHen Till solves:

### Blocked scraper
As you try to scale up the number of requests, quite often, the target websites will try to block your requests using Captcha, or throttling, or denying your request completely.
Especially if those websites are pretty popular, they would've taken enough steps to ensure that they don't get scraped.

### Scaling your scraping infrastructure
Scraping to millions or even billions of records requires a much more pre-planning. It's not simply as running your existing web scraper script in a bigger CPU/Ram machine. 
More thoughts are needed, such as: 

- How to do logging at scale
- Where to store the data
- The bandwidth used 
- Rotating through IP proxies
- File system load
- etc.

### Scraper Maintenance
Maintaining a highly scalable scrapers are not that easy. Because of the sheer number of requests that are being made, and the amount of interactions between your scrapers and the target websites, for it to run smoothly, you need to think through on how to maintain your scrapers on a regular basis. You need to know how to raise and triage errors as they occur on your scrapers, not all errors on web scraping should be treated equally. some are ignorable, and some are urgent. So, you will need to know what will be the details of your development-deployment-maintenance process will be.

### Postmortem analysis & reproducability
The biggest difficulty facing any web scraper developer is when there are scraping failures. Your scraper fails when fetching or parsing certain URLs, but when you look at the target website and URLs, everything look fine. How do you troubleshoot what already happened in scenario?. How do you reproduce that failed scrape so that you can fix the issue?

### Starting over from scratch when it fails mid-way
Websites change all the time, and without notice. Imagine running your web scraper for a week and then suddenly, somewhere along the way, it fails. It is frustrating that once you've fixed the scraper, there is a high chance that you'd need to start over from scratch again. And, on top of this, there are additional consequences, such as time delay, and further charges related to proxy usage, etc. 
## Features
### Automatic Retries
You don't have to write the retry logic in your scraper code, Till will retry your request up to 60 seconds (or however you wish). All you need to do is make sure that your scraper's timeout accomodates this.
### Managing Cookies
No need to build your own cookie management logic in your scraper codes. Till stores the cookies for you so that you can easily set and get the cookies on any request.

### User-Agent randomizer 
Till automatically generates random user-agent on every request. Choose to identify your scraper as a desktop browser, or a mobile browser, or you can even override it with your own custom user-agent.

### Proxy IP address rotation
Supply a list of proxy IPs, and Till will randomly use them on every request. Saves you time in needing to setup a separate proxy rotation service.

### Advanced Logging
Till will log your requests based on if it's a successful request (2XX status code) or failed request (non 2XX status code). This will allow you to easily troubleshoot your scraper later. You can also export the log in the [HAR](https://en.wikipedia.org/wiki/HAR_(file_format)) format, and you can open this in your Chrome's (or other browsers) inspector tool.

### HTTP Response caching
Till will cache all your HTTP responses (and their contents) locally, so that when you need run your scraper again, Till will reuse the same cached response and contents without needing do an actual request to the target server. You can even specify the freshness criteria of the cached contents to use. If the cache is outside of your freshness criteria, Till will send a real request to the target server and store that in the cache. 

### Unique Request Identifier 
Till uses [DataHen platform](https://www.datahen.com/platform)'s convention of marking every request with a unique signature (we call this the Global ID, or GID for short). Every request that is sent through Till, will return a response with the header `X-DH-GID` that contains the GID. This GID allows you to easily troubleshoot requests when you need to look up a specific requests in the log, or contents in the cache.


## How it works

Till runs as a standalone application that listens to incoming requests and proxies that requests to the target server as needed. While it does so, it logs and caches the requests.

Connect your scraper to Till via the `proxy` protocol that are typically common in any programming language.

Your scraper will then continue to run as-is and it will get instantly become more unblockable, scalable and maintainable.


## Usage

After installing, start Till with the following command
```bash
$ till start # this will run Till in port 2933
```

After you have started Till, your scraper code can then connect to Till via the `proxy` protocol.
### Curl

To connect using Curl, you can do the following

```bash
$ curl --proxy http://localhost:2933 https://fetchtest.datahen.com/echo/request
```
