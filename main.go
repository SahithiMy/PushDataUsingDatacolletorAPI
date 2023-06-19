package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	filePath := "2023-6-12-15.json"

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	url := "https://ec2-13-208-220-145.ap-northeast-3.compute.amazonaws.com/api/v0/events/data-collector"
	token := "yWTSCdCOQYSHVhJOLyQv6JSwakw="

	// Read the JSON file
	data := getData(filePath)

	structureComplianceReport(data, client, url, token)

}

func JSONReader(v interface{}) (r io.Reader, err error) {
	if debug_on() {
		jsonout, err := json.Marshal(v)
		if err != nil {
			fmt.Println("Error encoding JSON:", err)
		}
		fmt.Printf("\n\nJSON IN: %+v \n JSON ERR: %+v\n", string(jsonout), err)
	}
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(v)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
	r = bytes.NewReader(buf.Bytes())
	return
}

func debug_on() bool {
	return false
}

func getProfiles(data map[string]interface{}) map[string]interface{} {
	for _, profile := range data["profiles"].([]interface{}) {
		if profileMap, ok := profile.(map[string]interface{}); ok {
			for _, control := range profileMap["controls"].([]interface{}) {
				if controls, ok := control.(map[string]interface{}); ok {
					controls["tags"] = []string{}
					delete(controls, "tags")
				}
			}

		}
	}
	return data
}

func getEndTime(data map[string]interface{}) map[string]interface{} {
	if endtime, ok := data["end_time"].(map[string]interface{}); ok {
		name := endtime["seconds"]
		str := fmt.Sprintf("%v", name)
		intVal, err := strconv.ParseFloat(str, 64)
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		delete(data, "end_time")
		endTime := time.Unix(int64(intVal), 0)
		formattedTime := endTime.UTC().Format("2006-01-02T15:04:05Z")
		data["end_time"] = formattedTime

	}
	return data
}

func apiRequest(data map[string]interface{}, client *http.Client, url string, token string) *http.Response {

	var bodyJSON io.Reader = nil
	if data != nil {
		var err error
		// TODO: @afiune check panic!?
		bodyJSON, err = JSONReader(data)
		if err != nil {
			return nil
		}
	}
	request, err := http.NewRequest("POST", url, bodyJSON)

	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return nil
	}
	request.Header.Add("api-token", token)
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Accept", "*/*")

	// Send the request
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return nil
	}
	return response
}

func getData(filePath string) []map[string]interface{} {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return nil
	}
	defer jsonFile.Close()

	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return nil
	}
	// Set up the request
	var data []map[string]interface{}
	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON data: %v", err)
	}
	return data
}

func structureComplianceReport(data []map[string]interface{}, client *http.Client, url string, token string) {
	for i := range data {
		if data[i]["type"] == nil {
			data[i]["type"] = "inspec_report"
		}
		if data[i]["node_uuid"] == nil {
			data[i]["node_uuid"] = data[i]["node_id"]
		}
		if data[i]["report_uuid"] == nil {
			data[i]["report_uuid"] = data[i]["id"]
			delete(data[i], "id")
		}
		data[i] = getProfiles(data[i])
		data[i] = getEndTime(data[i])
		response := apiRequest(data[i], client, url, token)
		// Check the response
		if response.StatusCode == 200 {
			fmt.Println("DataCollector API call successful", response.Status, i)
		} else {
			fmt.Println("DataCollector API call failed", response.Status)
		}
	}
}
