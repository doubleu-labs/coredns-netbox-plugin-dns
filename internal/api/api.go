package api

import (
	"compress/gzip"
	"context"
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

func checkResp(r *http.Response) (err error) {
	if r.StatusCode != http.StatusOK {
		err = fmt.Errorf(
			"request error [%d] %q",
			r.StatusCode,
			r.Status,
		)
	}
	return
}

func getReader(r *http.Response) (rc io.ReadCloser, err error) {
	if r.Header.Get("Content-Encoding") == "gzip" {
		reader, readerErr := gzip.NewReader(r.Body)
		if readerErr != nil {
			err = fmt.Errorf("could not create gzip reader: %w", readerErr)
			return
		}
		rc = reader
		return
	}
	rc = r.Body
	return
}

func get[T any](ctx context.Context, c *Client, uri string) (out T, err error) {
	request, reqErr := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if reqErr != nil {
		err = reqErr
		return
	}
	response, respErr := c.Do(request)
	if respErr != nil {
		err = respErr
		return
	}
	defer closeBody(response.Body, &err, "closing response body")
	if rerr := checkResp(response); rerr != nil {
		err = rerr
		return
	}
	reader, readErr := getReader(response)
	if readErr != nil {
		err = readErr
		return
	}
	decoder := json.NewDecoder(reader)
	if decodeErr := decoder.Decode(&out); decodeErr != nil {
		err = fmt.Errorf("could not decode response: %w", decodeErr)
		return
	}
	return
}

func getMany[T any](ctx context.Context, c *Client, uri string) (
	out []T,
	err error,
) {
	nextUri := uri
	for nextUri != "" {
		responses, getErr := get[manyResp[T]](ctx, c, nextUri)
		if getErr != nil {
			err = getErr
			return
		}
		if out == nil {
			out = make([]T, 0, responses.Count)
		}
		out = slices.Concat(out, responses.Results)
		nextUri = responses.Next
	}
	return
}
