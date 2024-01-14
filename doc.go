/*
The go-mockhttp package extend standard net/http client library to provide a familiar HTTP client interface but with mock capabilities for testing.
It is a thin wrapper over the standard net/http client library and exposes nearly the same public API.
This makes mockhttp very easy to drop into existing programs.

The mock responses that will be returned by mockhttp client library is defined based on mock definitions.
Currently only support loading the mock definitions from files.

# Basics

How does go-mockhttp wraps the net/http client library ?

As Go http client library  use a *http.Client with definition that can be referred here: https://pkg.go.dev/net/http#Client.

Based of the documentation, the Client had 6 main method:

  - (c) CloseIdleConnections()

  - (c) Do(req)

  - (c) Get(url)

  - (c) Head(url)

  - (c) Post(url, contentType, body)

  - (c) PostForm(url, data)

Using this as the base reference, we could easily extend the standard Go http.Client struct into any custom struct that we want. To actually stub the 3rd party dependencies (via HTTP call), we could modify these method:

  - (c) Do(req)

  - (c) Get(url)

  - (c) Head(url)

  - (c) Post(url, contentType, body)

  - (c) PostForm(url, data)

that relates heavily on exchanging actual data to upstream service. Specifically, we apply this approach:

	=> Check if req match with loaded (in runtime) Mock Definition
	  => Yes? Use response defined in Mock Definition
	  => No?  Continue the requests to upstream service

# Mock Definitions

A term to describe a specification to determine how to match a request to the mock responses that defined using a file (as a `yaml` file) that includes:

  - Host, endpoint path and HTTP Method of upstream service that we want to mock.

  - Supported http requests format is JSON, XML, Form for POST, PUT, PATCH requests.

  - Description field that is used to describe what's the mock definition is.

  - Multiple (array) responses that can be used as the mock responses that match the `host`, `endpoint path` and `HTTP method` defined in the spec.

Example:

	host: marketplace.com
	path: /check-price
	method: POST
	desc: Testing Marketplace Price Endpoint
	responses:
	  - response_headers:
	    Content-Type: application/json
	    response_body: "{\"user_name\": \"Mocker\",\r\n \"price\": 1000}"
	    status_code: 200
	  - response_headers:
	    Content-Type: application/json
	    response_body: "{\"user_name\": \"William\",\r\n \"price\": 2000}"
	    delay: 1000
	    status_code: 488
	    enable_template: false
	    rules:
	  - body.name == "William"

There are 3 ways on how the library will try to match the endpoint path:

 1. Exact Match: /v1/api/mock/1

 2. With Path Param: /v1/api/mock/:id

 3. Wildcard: /v1/api/*

What happen when the request have no matching Mock Definition?

There are 2 conditions that might happen:

 1. Request don't match host, path, and method => http client will immediately call actual upstream service

 2. Request match host, path, and method, but didn't satisfy the rules in responses => will try to use default response (response with no rules). If no default response defined, will simply call actual upstream service.

# Example Usage

Here are the example on how to use the library:

	resolver, err := mockhttp.NewFileResolverAdapter(definitionDirPath)
	if err != nil {
	  panic(err)
	}

	err = resolver.LoadDefinition(context.Background())
	if err != nil {
	  panic(err)
	}

	mockClient := mockhttp.NewClient(resolver)
	resp, err := .Get("/foo")
	if err != nil {
	  panic(err)
	}
*/

package mockhttp
