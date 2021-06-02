# client-go-multidialer

_client-go says: "just give me a working TCP connection, I will do the REST"_

This provides a new constructor to client-go with a custom Dialer that offers high availability by providing a working connection, if possible, to any of the apiservers in the cluster.

It needs to be initialized with one working apiserver IP that will be the seed, once it connects it will obtain the rest of apiserver endpoints.

## Usage

The instantiation is similar to the current client-go constructor, with the addition of one context parameter:

```go
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientset, err := client.NewForConfig(ctx, config)
	if err != nil {
		panic(err.Error())
	}
```
 
## How it works

It provides a custom dialer to the client-go client, it also wraps the actual dialer in case client-go was already using one, like in the Kubelet case.

The custom dialer implement the following logic:

1. Connects to the provided endpoint
2. Spawn a go routine that queries the apiserver to the get current available endpoints, these are available under the special endpoint object "kubernetes" on the "default" namespace
3. When the http transport in client-go ask for a connection to the dialer:
	- tries to connect to the last working apiserver endpoint, if it fails
	- tries to connect to the other endpoints, if it succeeds returns the connection and store the endpoint as the one to be used next time
	- if none of the endpoints work, it falls back to the initial configured endpoint

The fact that it gathers the apiserver endpoints periodically allows to change the apiserver IPs, shrink or grow the cluster.

## Implementation

The implementation is composed of:

- a [client-go constructor](./client/client.go): It returns a client-go object with the multidialer
- a [custom dialer](./multidialer/multidialer.go): Implements the multiple dialer logic, uses a resolver to get a list of available apiservers.
- an [apiserver resolver](./multidialer/resolver.go): Implements the apiserver endpoints resolvers

## Examples
 
There are two examples on how to use it to create a client-go:

- as a client to query the kubernetes API https://github.com/aojea/client-go-multidialer/blob/main/_example/client/main.go
- as a client to create a controller https://github.com/aojea/client-go-multidialer/blob/main/_example/controller/main.go


