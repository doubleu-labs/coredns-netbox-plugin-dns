package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

var (
	apiPath       = [3]string{"api", "plugins", "netbox-dns"}
	apiPathString = strings.Join(apiPath[:], "/")
)

// JoinAPIPath takes the url.URL of the Netbox instance and appends the
// path for the DNS plugin API.
func JoinAPIPath(u *url.URL) *url.URL {
	return u.JoinPath("api", "plugins", "netbox-dns")
}

type manyResp[T any] struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []T    `json:"results"`
}

func closeBody(c io.Closer, err *error, msg string) {
	if closerErr := c.Close(); closerErr != nil {
		closerErr = fmt.Errorf("%s: %w", msg, closerErr)
		if *err != nil {
			*err = errors.Join(*err, closerErr)
			return
		}
		*err = closerErr
	}
}

func checkResp(r *http.Response) error {
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
	request, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return out, err
	}
	response, err := c.Do(request)
	if err != nil {
		return out, err
	}
	defer closeBody(response.Body, &err, "closing response body")
	if rerr := checkResp(response); rerr != nil {
		return out, rerr
	}
	decoder := json.NewDecoder(response.Body)
	if decodeErr := decoder.Decode(&out); decodeErr != nil {
		return out, fmt.Errorf("could not decode response: %w", decodeErr)
	}
	return out, nil
}

func getMany[T any](c *Client, uri string) ([]T, error) {
	var out []T
	nextUri := uri
	for nextUri != "" {
		responses, err := get[manyResp[T]](c, nextUri)
		if err != nil {
			return out, err
		}
		if out == nil {
			out = make([]T, 0, responses.Count)
		}
		out = slices.Concat(out, responses.Results)
		nextUri = responses.Next
	}
	return out, nil
}
