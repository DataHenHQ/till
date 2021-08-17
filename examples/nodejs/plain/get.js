const got = require('got');
const {HttpsProxyAgent} = require('hpagent');

(async function main() {
  const response = await got.get('https://fetchtest.datahen.com/echo/request', {
    agent: {
      https: new HttpsProxyAgent({
        proxy: 'http://localhost:2933',
      }),
    },
    https: {
      rejectUnauthorized: false,
    },
    headers: {
      'X-DH-Cache-Freshness': 'now' // Forces a cache miss.
    }
  });

  console.log({ response });
})();