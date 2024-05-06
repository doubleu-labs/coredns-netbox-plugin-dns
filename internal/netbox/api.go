package netbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type APIRequestClient struct {
	Client    *http.Client
	NetboxURL *url.URL
	Token     string
	UserAgent string
}

type APIResultModel interface {
	Record | Zone
}

type APIManyResponse[T APIResultModel] struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []T    `json:"results"`
}

func doGet(
	requestClient *APIRequestClient,
	url string,
) (*http.Response, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set(
		"Authorization",
		fmt.Sprintf("Token %s", requestClient.Token),
	)

	request.Header.Set("User-Agent", requestClient.UserAgent)

	return requestClient.Client.Do(request)
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

func get[T APIResultModel](
	requestClient *APIRequestClient,
	url string,
) (T, error) {
	var out T
	response, err := doGet(requestClient, url)
	if err != nil {
		return out, err
	}
	defer response.Body.Close()
	if err := responseError(response); err != nil {
		return out, err
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&out); err != nil {
		return out, fmt.Errorf("could not unmarshal response: %w", err)
	}
	return out, nil
}

func getMany[T APIResultModel](
	requestClient *APIRequestClient,
	url string,
) ([]T, error) {
	nextUrl := url
	var out []T

	for nextUrl != "" {
		response, err := doGet(requestClient, nextUrl)
		if err != nil {
			return out, err
		}
		defer response.Body.Close()

		if err := responseError(response); err != nil {
			return out, err
		}

		var apiResponse APIManyResponse[T]
		decoder := json.NewDecoder(response.Body)
		if err := decoder.Decode(&apiResponse); err != nil {
			return out, fmt.Errorf("could not unmarshal response: %w", err)
		}

		if out == nil {
			out = make([]T, 0, apiResponse.Count)
		}
		out = append(out, apiResponse.Results...)

		nextUrl = apiResponse.Next
	}

	return out, nil
}
