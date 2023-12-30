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

func main() {

	resolver, err := mockhttp.NewFileResolverAdapter("./example/mock-data")
	if err != nil {
		panic(err)
	}

	err = resolver.LoadPolicy(context.Background())
	if err != nil {
		panic(err)
	}

	mockClient := mockhttp.NewClient(resolver)
	mockClient.StandardClient().Timeout = 1 * time.Minute

	reqBody := `{"va": "706081274966275"}`
	req, _ := http.NewRequest(http.MethodPost, "http://google.com/inquiry", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Add("Content-Type", "application/json")
	client := mockClient.StandardClient()
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
