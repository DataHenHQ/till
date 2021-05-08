package tillclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/DataHenHQ/license"
	"github.com/go-resty/resty/v2"
)

var BaseURL string

type service struct {
	client *Client
}

type Client struct {
	*resty.Client
	Token string

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	Instances *InstancesService
}

func NewClient(token string) (c *Client, err error) {

	c = &Client{
		Client: resty.New(),
	}
	c.SetTimeout(1 * time.Minute)
	c.SetHostURL(BaseURL)
	c.Token = token

	c.OnAfterResponse(verifySignature)

	c.common.client = c

	// assigns the common service
	c.Instances = (*InstancesService)(&c.common)

	return c, nil
}

// a middleware that verifies the signature from the api response, and then replace it with the actual content data
func verifySignature(c *resty.Client, resp *resty.Response) error {

	// if error status code is more than 399, return an error
	if resp.StatusCode() > 399 {
		return &CustomError{
			StatusCode: resp.StatusCode(),
			Err:        errors.New(resp.Status()),
		}
	}

	// verify the response body and extract the data
	data, err := license.Verify(resp.Body())
	if err != nil {
		return err
	}

	// replace the raw response body with the new content data.
	// NOTE: we can't use resp.Body() anymore in downstream, because it still refers to old body
	nbody := ioutil.NopCloser(bytes.NewReader(data))
	resp.RawResponse.Body = nbody

	return nil // if its success otherwise return error
}

func (c *Client) NewRequest(ctx context.Context, urlStr string, body interface{}) (*resty.Request, error) {
	if c.Token == "" {
		return nil, errors.New("token required")
	}

	req := c.R()
	req.SetContext(ctx)

	req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	return req, nil
}
