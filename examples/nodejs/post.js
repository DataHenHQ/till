const got = require('got');
const {HttpsProxyAgent} = require('hpagent');

(async function main() {
  const response = await got.post('https://postman-echo.com/post', {
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
    },
    json: {
        hello: 'world'
    }
  });

  console.log({ response });
})();
