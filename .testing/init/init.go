package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

var (
	apiRoot = "http://localhost:9999/api/plugins/netbox-dns"
	token   = "w5pgWXPqZVmngLN4w4XwuPvZfUC72ytDxnnHgEmI"
	execdir string
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get current filename")
	}
	execdir = filepath.Dir(filename)
}

func post(client *http.Client, path string, filepath string) (string, []byte) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	stat, _ := file.Stat()
	req, err := http.NewRequest("POST", apiRoot+path, file)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json; indent=4")
	req.ContentLength = stat.Size()

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return resp.Status, content
}

func main() {
	nameservers := filepath.Join(execdir, "nameservers.json")
	zones := filepath.Join(execdir, "zones.json")
	records := filepath.Join(execdir, "records.json")
	client := &http.Client{}

	nsStatus, nsContent := post(client, "/nameservers/", nameservers)
	log.Printf("nameservers: %s\n%s", nsStatus, nsContent)

	zoneStatus, zoneContent := post(client, "/zones/", zones)
	log.Printf("zones: %s\n%s", zoneStatus, zoneContent)

	recordStatus, recordContent := post(client, "/records/", records)
	log.Printf("records: %s\n%s", recordStatus, recordContent)
}
