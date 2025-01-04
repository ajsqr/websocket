# websocket

websocket is an external dependency free, pure go implementation of [RFC-6455](https://datatracker.ietf.org/doc/html/rfc6455). This project is currently in its [alpha stage](https://en.wikipedia.org/wiki/Software_release_life_cycle) and the APIs can break in new releases. 

## Getting started

### Prerequisites
websocket requires Go version 1.23 or above.

### Getting websocket
With Go's module support, go [build|run|test] automatically fetches the necessary dependencies when you add the import in your code:

```go
import "github.com/ajsqr/websocket"
```

Alternatively, use go get:

```go
go get -u github.com/ajsqr/websocket"
```

## Running websocket

A basic example:

```go
package main

import (
	 "github.com/ajsqr/websocket"
	 "github.com/go-chi/chi/v5"
	 "net/http"
	 "time"
	 "fmt"
)

var opener = websocket.WSOpener{
	MaxBytes: 20,
}

func hello(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ws, err := opener.Open(w,r, websocket.TextWebsocket)
	if err != nil{
		panic(err.Error())
	}

	// wait for a message from the client
	clientHello, err := ws.Receive(ctx)
	if err != nil{
		panic(err.Error())
	}

	fmt.Printf("client says : %s", string(clientHello))

	for {
		err := ws.Send(ctx, []byte("hello"))
		if err != nil{
			panic(err.Error())
		}

		time.Sleep(5*time.Second)
	}

}

func main() {
	router := chi.NewRouter()
	router.Get("/", hello)
	err := http.ListenAndServe(":8123", router)
	if err != nil{
		panic(err.Error())
	}
}
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first
to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)