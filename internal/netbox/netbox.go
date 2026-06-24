package netbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type manyResponse[T any] struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []T    `json:"results"`
}

func closeResponseBody(closer io.Closer, err *error, c string) {
	if closeErr := closer.Close(); closeErr != nil {
		closeErr = fmt.Errorf("%s: %w", c, closeErr)
		if *err != nil {
			*err = errors.Join(*err, closeErr)
			return
		}
		*err = closeErr
	}
}

func responseError(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"request error [%d] %q",
			r.StatusCode,
			r.Status,
		)
	}

	return nil
}

func get[T any](c *Client, uri string) (T, error) {
	var out T

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token.raw))
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.Client.Do(req)
	if err != nil {
		return out, err
	}

	defer closeResponseBody(resp.Body, &err, "response body")
	if err := responseError(resp); err != nil {
		return out, err
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return out, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return out, nil
}

func getMany[T any](c *Client, uri string) ([]T, error) {
	nextUri := uri
	var out []T

	for nextUri != "" {
		resp, err := get[manyResponse[T]](c, nextUri)
		if err != nil {
			return nil, err
		}

		if out == nil {
			out = make([]T, 0, resp.Count)
		}
		out = append(out, resp.Results...)

		nextUri = resp.Next
	}

	return out, nil
}
