package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/buger/jsonparser"
)

const (
	listUrl = "https://api.cloudflare.com/client/v4/zones/%s/dns_records" // zone_identifier
	deleteUrl = "https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s" // && dns_record identifier
)

var (
	zone_identifier string
	X_Auth_Key string
	X_Auth_Email string
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func load() {
	log.Println("Loading config.txt")
	txt, err:= os.ReadFile("config.txt")
	checkErr(err)

	txt = bytes.ReplaceAll(txt, []byte("\r\n"), []byte("\n"))
	config := bytes.Split(txt, []byte("\n"))

	for _, v := range config {
		// log.Println("v:", string(v))
		if bytes.Contains(v, []byte("email")) {
			X_Auth_Email = string(v[6:])
		}
		if bytes.Contains(v, []byte("key")) {
			X_Auth_Key = string(v[4:])
		}
		if bytes.Contains(v, []byte("zone_id")) {
			zone_identifier = string(v[8:])
		}
	}

	log.Println("Email:", X_Auth_Email)
	log.Println("Key:", X_Auth_Key)
	log.Println("Zone ID:", zone_identifier)
}

func getList() []byte {
	req, err := http.NewRequest("GET", fmt.Sprintf(listUrl, zone_identifier), nil)
	req.Header.Set("X-Auth-Key", X_Auth_Key)
	req.Header.Set("X-Auth-Email", X_Auth_Email)
	checkErr(err)

	resp, err := (&http.Client{}).Do(req)
	checkErr(err)

	body, err := io.ReadAll(resp.Body)
	checkErr(err)
	
	if resp.StatusCode != 200 {
		log.Fatal("Get list dns records failed.")
	}
	return body
}

func getCount() int64 {
	body := getList()

	total_count, err := jsonparser.GetInt(body, "result_info", "total_count")
	checkErr(err)
	return total_count
}

func getRecords() []byte {
	body := getList()

	records, _, _, err := jsonparser.Get(body, "result")
	checkErr(err)

	// log.Println("Length of records:", len(records))
	// log.Println("Records:", string(records))
	return records
}

func getIds(records []byte) []string {
	ids := make([]string, 0)
	jsonparser.ArrayEach(records, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		checkErr(err)
		id, err := jsonparser.GetString(value, "id")
		checkErr(err)
		ids = append(ids, id)
	})
	return ids
}

var deletedCount int

func delete(i, len int, id string) {
	go func() {
		req, err := http.NewRequest("DELETE", fmt.Sprintf(deleteUrl, zone_identifier, id), nil)
		checkErr(err)
		req.Header.Set("X-Auth-Key", X_Auth_Key)
		req.Header.Set("X-Auth-Email", X_Auth_Email)

		resp, err := (&http.Client{}).Do(req)
		checkErr(err)

		if resp.StatusCode != 200 {
			log.Fatal("Delete id: ", id, " failed.")
		}
		deletedCount ++
		log.Println("Deleted record id", id, "successfully.", fmt.Sprintf("%d/%d", deletedCount, len))
	}()
}

func deleteRecords(ids []string) {
	deletedCount = 0
	for i, id := range ids {
		delete(i, len(ids), id)
		// time.Sleep(time.Microsecond * 20)
	}
	for deletedCount != len(ids) {
		time.Sleep(time.Second)
	}
}

func main() {
	load()

	count := getCount()
	log.Println("Total count:", count)

	for count != 0 {
		ids := getIds(getRecords())
		log.Println("Length of ids to delete:", len(ids))
		deleteRecords(ids)
		log.Println("Deleted", len(ids), "records successfully.")
		log.Println("Rest of records count", getCount())
	}
	log.Println("Done.")
	fmt.Scan()
}