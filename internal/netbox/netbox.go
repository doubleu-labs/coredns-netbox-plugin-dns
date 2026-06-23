package netbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	Client    *http.Client
	NetboxURL *url.URL
	Token     string
	UserAgent string
}

type ManyResponse[T any] struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []T    `json:"results"`
}

func closeResponseBody(body io.Closer, err *error, context string) {
	if closeErr := body.Close(); closeErr != nil {
		closeErr = fmt.Errorf("%s: %w", context, closeErr)
		if *err != nil {
			*err = errors.Join(*err, closeErr)
			return
		}
		*err = closeErr
	}
}

func responseError(response *http.Response) error {
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"request error [%d] %q",
			response.StatusCode,
			response.Status,
		)
	}

	return nil
}

func get[T any](
	client *Client,
	url string,
) (T, error) {
	var out T

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return out, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.Token))
	request.Header.Set("User-Agent", client.UserAgent)
	response, err := client.Client.Do(request)
	if err != nil {
		return out, err
	}

	defer closeResponseBody(response.Body, &err, "response body")
	if err := responseError(response); err != nil {
		return out, err
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&out); err != nil {
		return out, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return out, nil
}

func getMany[T any](
	client *Client,
	url string,
) ([]T, error) {
	nextUrl := url
	var out []T

	for nextUrl != "" {
		resp, err := get[ManyResponse[T]](client, nextUrl)
		if err != nil {
			return nil, err
		}

		if out == nil {
			out = make([]T, 0, resp.Count)
		}
		out = append(out, resp.Results...)

		nextUrl = resp.Next
	}

	return out, nil
}
