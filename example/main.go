package main

import (
	"fmt"
	"time"

	"github.com/William9923/go-mockhttp"
)

func main() {
	mockClient := mockhttp.NewClient(mockhttp.DefaultResolver())
	mockClient.Delay = 100 * time.Millisecond
	mockClient.StandardClient().Timeout = 1 * time.Minute

	resp, err := mockClient.Get("http://google.com")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.StatusCode)
}
