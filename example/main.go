package main

import (
	"context"
	"fmt"
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

	resp, err := mockClient.Get("http://google.com")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.StatusCode)
}
