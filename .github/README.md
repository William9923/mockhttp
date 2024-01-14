# go-mockhttp

<a href="https://pkg.go.dev/github.com/William9923/go-mockhttp">
    <img src="https://pkg.go.dev/badge/github.com/William9923/go-mockhttp.svg">
</a>
<a href="https://goreportcard.com/report/github.com/william9923/go-mockhttp">
    <img src="https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat">
</a>
<a href="https://github.com/William9923/go-mockhttp/actions/workflows/mockhttp.yaml">
    <img src="https://github.com/William9923/go-mockhttp/actions/workflows/mockhttp.yaml/badge.svg?branch=master">
</a>

### Overview

#### Problem

- You have http client in your Go code, and want to start testing various case on it.
- For unit test, you can mock those response with [httptest](https://pkg.go.dev/net/http/httptest)
- What if your QA guys asked you various edge case for QA testing / for integration test that tied with external service.
  It might be easy if we had full control of the upstream service, but what if don't have it ?
  Sometimes, we might need to prepare various mocks for testing edge cases to ensure everything works as expected.

  So, what to do? Should we... :

  - Hardcoded many if statement in the codebase to prepare those edge case ?
  - Build a sandbox service to represent 3rd party / external upstream service ?
  - Give up and don't test those case at all ?

While all 3 options is definitely possible (except the last one 😠), introducing `go-mockhttp`...

**go-mockhttp** is a testing layer for Go standard [http](https://pkg.go.dev/net/http) client library. It allows stubbed responses to be configured for matched HTTP requests that had been defined and can be used to test your application's service layer in unit test or even in actual test server for many various case that depends on 3rd party responses.

It use **Mock Definition**, a term that we use to define a specification of:

- How to determine (match) whether a request should be mock / not
- How to determine (match) which mock response should be used, based on request entity (using flexible rules)
- Other misc thing that might be useful for testing

Equipped with these capabilities, now the `*http.Client` that you use can be extended to also supports integration / automation / manual testing easily.

**WARNING!** While you can definitely use it on production, it is suggested to only use this for helping testing purposes.

#### How it works?

As Go http client library (from Go [net/http](https://pkg.go.dev/net/http) client library) use a `*http.Client` with definition that can be referred [here](https://pkg.go.dev/net/http#Client).

Based of the documentation, the `Client` had 6 main method:

- `(c) CloseIdleConnections()`
- `(c) Do(req)`
- `(c) Get(url)`
- `(c) Head(url)`
- `(c) Post(url, contentType, body)`
- `(c) PostForm(url, data)`

Using this as the base reference, we could easily extend the standard Go http.Client struct into any custom struct that we want. To actually stub the 3rd party dependencies (via HTTP call), we could modify these method:

- `(c) Do(req)`
- `(c) Get(url)`
- `(c) Head(url)`
- `(c) Post(url, contentType, body)`
- `(c) PostForm(url, data)`

that relates heavily on exchanging actual data to upstream service. Specifically, we apply this approach:

```
=> Check if req match with loaded (in runtime) Mock Definition
   => Yes? Use response defined in Mock Definition
   => No?  Continue the requests to upstream service
```

### Get Started

Version 0.2.1 and before are requiring Go version 1.13+.

#### Installation

```bash
go get github.com/William9923/go-mockhttp@latest
```

#### Examples

Using this library should look almost identical to what you would do with net/http. The most simple example of a GET request is shown below:

```go
...

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

...
```

The returned response object is an \*http.Response, the same thing you would usually get from net/http. Had the request match with the **Mock Definition** loaded into `*mockhttp.Client`, the above call would instead be stubbed with response defined in **Mock Definition**.

### Roadmap

What will the library try to improve in the future?

- As of now, the mock **delay** haven't been able to be integrated in the
  library. It will be the utmost priority for next version!
- Provide more example for easier adoption of the library in any existing projects.
- Additional adapter supports (inspired by [casbin](https://casbin.org/docs/adapters)), to allow more ways to load **Mock Definition** from different storage.
- Extending ways to use **Mock Definition** in other language (not only Go), as **Mock Definition** can be used cross-language.
- Build mockhttp as a service instead of a library, to accomodate for non-Go service that would like to utilize it (**Mock as a Service**).

### Limitations!

- Due to `go-mockhttp` usage that intervene directly of any http call, it is
  not advised to use the http.Client created by `go-mockhttp` to PRODUCTION!.
  You can find workaround on this by providing flags or by config to determine
  whether to use standard `net/http` http client or `go-mockhttp` http client
  during app / client initialization.
- `go-mockhttp` currently only support mock based on file (via mock definition
  files). Other datastore to load the mock definition still haven't supported!

### FAQ

#### How to use with stdlib \*http.Client ?

Similar to [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp/), It's possible to fully convert `*mockhttp.Client` directly into a `http.Client`. This makes adoption `mockhttp` is applicable in many situation with minimal effort. Simply configure a \*retryablehttp.Client as you wish, and then call StandardClient():

```go
...
  mockClient := mockhttp.NewClient(resolver) // assuming resolver for mock definition had been defined...
  resp, err := mockClient.Get("/foo")
  if err != nil {
    panic(err)
  }
...
```

#### What is Mock Definition ?

A term to describe a specification (as a `yaml` file) that includes:

- `Host`, `endpoint path` and `HTTP Method` of upstream service that we want to mock.
- Supported http requests format is `JSON`, `XML`, `Form` for `POST`, `PUT`, `PATCH`, `DELETE` requests.
- Description field that is used to describe what's the mock definition is.
- Multiple (array) responses that can be used as the mock responses that match the `host`, `endpoint path` and `HTTP method` defined in the spec.
- Each responses can includes:

  - `response_headers`: map of <string, string>
  - `response_body` : support all serializeable format (as string)
  - `status_code`: int
  - `enable_template`: allow templating for response_body, using request body information
  - `delay`: integer. Use milliseconds, to delay the responses before returning the response. Useful for testing timeout requests (context deadline).
  - `rules`: array of CEL expression that use request body information to evaluate the expression.

**Example:**

```yaml
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
```

**Match Behavior:**

There are 3 ways on how the library will try to match the endpoint path:

1. **Exact Match:** `/v1/api/mock/1`
2. **With Path Param:** `/v1/api/mock/:id`
3. **Wildcard:** `/v1/api/*`

**What happen when the request have no matching Mock Definition?**

There are 2 conditions that might happen:

1. Request don't match `host`, `path`, and `method` => http client will immediately call actual upstream service
2. Request match `host`, `path`, and `method`, but didn't satisfy the rules in responses => will try to use default response (response with no rules). If no default response defined, will simply call actual upstream service.

### License
Distributed under the MIT License. See LICENSE.txt for more information.
