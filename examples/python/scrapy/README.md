# Integrating Till with Scrapy

The example in [this directory](tutorial/) was taken from Scrapy's [tutorial page](https://docs.scrapy.org/en/latest/intro/tutorial.html) and modified to integrate with Till.

To integrate with Till, we only need to do two things:

1. On [middlewares.py](tutorial/tutorial/middlewares.py#L14) file, add the `TillMiddleware` class.
2. On [settings.py](tutorial/tutorial/settings.py#L54) file, enable the downloader middlewares and add the `tutorial.middlewares.TillMiddleware` key.

## 1. Install Till
Follow the instructions to [install Till](https://till.datahen.com/docs/installation)

## 2. Run the example

```bash
# Install Scrapy
$ pip install scrapy
# On the tutorial directory, run:
$ scrapy crawl quotes
```

### 3. Verify that it works

Visit the Till UI at [http://localhost:2980/requests](http://localhost:2980/requests) to see that your new requests are shown.