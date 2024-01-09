# go-mockhttp

[![build and test](https://github.com/William9923/go-mockhttp/actions/workflows/mockhttp.yaml/badge.svg?branch=master)](https://github.com/William9923/go-mockhttp/actions/workflows/mockhttp.yaml)

TODO:

- Make a better representation (image) , can be using tldraw alone => or if you find ways to illustrate via gif, even better => https://github.com/dai-shi/excalidraw-animate
- Make badge for the CI result => DONE
- Make Mock Definition specification
- Extract example into full fledge example golang project => let's try find some good example
- Future works => refer casbin, essentially, want more ways to provide mocks adapter (database, redis, etc...). Also, exploring ways to create rule even more flexible. Mock server for even more language agnostic approach for stubbing upstream service. => DONE

A gif / representation

<Table of content here bro...>

### Overview

**mockhttpclient** is a testing layer for Go standard [http](https://pkg.go.dev/net/http) client library. It allows stubbed responses to be configured for matched HTTP requests that had been defined and can be used to test your application's service layer in unit test or even in actual test server for many various case that depends on 3rd party responses.

**WARNING!** Don't use it on production, only for helping testing purposes.

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

Version 0.1.0 and before are requiring Go version 1.13+.

#### Installation

```bash
go get github.com/William9923/go-mockhttp
```

#### Examples

Using this library should look almost identical to what you would do with net/http. The most simple example of a GET request is shown below:

```go
...

  resolver, err := mockhttp.NewFileResolverAdapter(policyDirPath)
  if err != nil {
    panic(err)
  }

  err = resolver.LoadPolicy(context.Background())
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

- Additional adapter supports (inspired by [casbin](https://casbin.org/docs/adapters)), to allow more ways to load **Mock Definition** from different storage.
- Exploring ways to make the **Mock Definition** more re-useable, less restrictive and more expressive for the user.
- Build mockhttp as a service instead of a library, to accomodate for non-Go service that would like to utilize it (**Mock as a Service**).

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

**Specification:**

**Match Behavior:**

**What happen when no matching Mock Definition?**

**Example:**

TODO: change policy term => definition

### License

This project is under the MIT license.
