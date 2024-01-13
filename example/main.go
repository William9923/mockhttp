package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/William9923/go-mockhttp"
)

func buildMockHTTPClient(policyDirPath string) (*http.Client, error) {

	var (
		err error
	)

	resolver, err := mockhttp.NewFileResolverAdapter(policyDirPath)
	if err != nil {
		return nil, err
	}

	err = resolver.LoadPolicy(context.Background())
	if err != nil {
		return nil, err
	}

	mockClient := mockhttp.NewClient(resolver)
	mockClient.StandardClient().Timeout = 1 * time.Minute

	return mockClient.StandardClient(), nil
}

func main() {
	reqBody := `{"va": "706081274966275"}`
	req, _ := http.NewRequest(http.MethodPost, "http://google.com/inquiry", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Add("Content-Type", "application/json")
	client, err := buildMockHTTPClient("./mock-data")
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.StatusCode)
	body, _ := extractBody(resp)
	fmt.Println(body)
}

func extractBody(resp *http.Response) (string, error) {
	// Read the request body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Convert the body to a string
	bodyString := string(body)
	return bodyString, nil
}
