const puppeteer = require('puppeteer');

(async () => {
  const browser = await puppeteer.launch(
    {
      headless: true,
      ignoreHTTPSErrors: true, 
      acceptInsecureCerts: true, 
       args: [
         '--proxy-server=http://localhost:2933',
         '--ignore-certificate-errors',
         '--ignore-certificate-errors-spki-list ',
      ],
     }
  );

  const page = await browser.newPage();

  await page.setExtraHTTPHeaders({
    // Add the header to force a Cache Miss on Till
    'X-DH-Cache-Freshness': 'now' 
})

  await page.goto('https://fetchtest.datahen.com/echo/request');
  
  const txt = await page.content()
  console.log(txt);

  await browser.close();
})();