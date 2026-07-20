package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

var (
	hostAPIRoot        = "http://127.0.0.1:9999/api"
	pluginAPIRoot      = hostAPIRoot + "/plugins/netbox-dns"
	applicationJSON    = "application/json"
	token              string
	testDataViewsPaths []string
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get current filename")
	}
	testDataDir := filepath.Clean(
		filepath.Join(
			filepath.Dir(filename),
			"..",
			"testdata",
		),
	)
	testDataViewsPaths = []string{
		filepath.Join(testDataDir, "internal"),
		filepath.Join(testDataDir, "offsite"),
		filepath.Join(testDataDir, "public"),
	}
}

func provisionToken(client *http.Client) {
	payload := []byte(`{"username":"admin","password":"admin"}`)
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/users/tokens/provision/", hostAPIRoot),
		bytes.NewBuffer(payload),
	)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", applicationJSON)
	req.Header.Set("Accept", applicationJSON)
	req.ContentLength = int64(len(payload))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rc io.ReadCloser) {
		if closeErr := rc.Close(); closeErr != nil {
			log.Fatal(closeErr)
		}
	}(resp.Body)

	var tokenResponse struct {
		Key   string `json:"key"`
		Token string `json:"token"`
	}
	decodeErr := json.NewDecoder(resp.Body).Decode(&tokenResponse)
	if decodeErr != nil {
		log.Fatal(decodeErr)
	}
	token = fmt.Sprintf("nbt_%s.%s", tokenResponse.Key, tokenResponse.Token)
}

func post(c *http.Client, path string, filepath string) (string, string) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		closeErr := f.Close()
		if errors.Is(closeErr, fs.ErrClosed) {
			return
		}
		if closeErr != nil {
			log.Fatal(closeErr)
		}
	}(file)

	stat, statErr := file.Stat()
	if statErr != nil {
		log.Fatal(statErr)
	}
	req, reqErr := http.NewRequest(
		"POST",
		fmt.Sprintf("%s%s", pluginAPIRoot, path),
		file,
	)
	if reqErr != nil {
		log.Fatal(reqErr)
	}
	req.Header.Set("Content-Type", applicationJSON)
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))
	req.ContentLength = stat.Size()

	resp, respErr := c.Do(req)
	if respErr != nil {
		log.Fatal(respErr)
	}
	defer func(rc io.ReadCloser) {
		if closeErr := rc.Close(); closeErr != nil {
			log.Fatal(closeErr)
		}
	}(resp.Body)

	content, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}
	return resp.Status, string(content)
}

func main() {
	client := &http.Client{}
	provisionToken(client)

	for tdvp := range slices.Values(testDataViewsPaths) {
		testFiles, err := os.ReadDir(tdvp)
		if err != nil {
			log.Fatal(err)
		}
		for testFile := range slices.Values(testFiles) {
			filename := strings.TrimSuffix(testFile.Name()[2:], ".json")
			viewDir := filepath.Base(tdvp)

			var apiPath string
			switch filename {
			case "views":
				apiPath = "/views/"
			case "nameservers":
				apiPath = "/nameservers/"
			case "zones":
				apiPath = "/zones/"
			case "records":
				apiPath = "/records/"
			default:
				panic(fmt.Sprintf("unknown test file: %s", testFile.Name()))
			}
			status, content := post(
				client,
				apiPath,
				filepath.Join(tdvp, testFile.Name()),
			)
			log.Printf("%s/%s: %s\n", viewDir, testFile.Name(), status)
			log.Printf("%s\n", content)
		}
	}
}
