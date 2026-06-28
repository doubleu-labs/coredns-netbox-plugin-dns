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
)

var (
	hostApiRoot = "http://localhost:9999/api"
	apiRoot     = hostApiRoot + "/plugins/netbox-dns"
	token       string
	testdataDir string
)

type tokenProvisionResponse struct {
	Key   string `json:"key"`
	Token string `json:"token"`
}

func (t tokenProvisionResponse) String() string {
	return fmt.Sprintf("nbt_%s.%s", t.Key, t.Token)
}

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get current filename")
	}
	testdataDir = filepath.Clean(
		filepath.Join(filepath.Dir(filename), "..", "..", "testdata"),
	)
}

func closeBody(body io.ReadCloser) {
	err := body.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func provisionToken(client *http.Client) {
	payload := []byte(`{"username": "admin", "password": "admin"}`)
	req, err := http.NewRequest(
		"POST",
		hostApiRoot+"/users/tokens/provision/",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	var tokenProvisionResponse tokenProvisionResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenProvisionResponse)
	if err != nil {
		closeBody(resp.Body)
		log.Fatal(err)
	}
	closeBody(resp.Body)
	token = tokenProvisionResponse.String()
}

func closeFile(file *os.File) {
	err := file.Close()
	if errors.Is(err, fs.ErrClosed) {
		return
	}
	if err != nil {
		log.Fatal(err)
	}
}

func post(client *http.Client, path string, filepath string) (string, []byte) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer closeFile(file)

	stat, _ := file.Stat()
	req, err := http.NewRequest("POST", apiRoot+path, file)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json; indent=4")
	req.ContentLength = stat.Size()

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer closeBody(resp.Body)

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return resp.Status, content
}

func main() {
	views := filepath.Join(testdataDir, "views.json")
	nameservers := filepath.Join(testdataDir, "nameservers.json")
	zones := filepath.Join(testdataDir, "zones.json")
	records := filepath.Join(testdataDir, "records.json")
	client := &http.Client{}

	provisionToken(client)

	viewsStatus, viewsContent := post(client, "/views/", views)
	log.Printf("views: %s\n%s", viewsStatus, viewsContent)

	nsStatus, nsContent := post(client, "/nameservers/", nameservers)
	log.Printf("nameservers: %s\n%s", nsStatus, nsContent)

	zoneStatus, zoneContent := post(client, "/zones/", zones)
	log.Printf("zones: %s\n%s", zoneStatus, zoneContent)

	recordStatus, recordContent := post(client, "/records/", records)
	log.Printf("records: %s\n%s", recordStatus, recordContent)
}
