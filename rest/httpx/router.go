package httpx

import "net/http"

// Router interface represents a http router that handles http requests.
type Router interface {
	http.Handler
	Handle(method, path string, handler http.Handler) error
	SetNotFoundHandler(handler http.Handler) //没有method+reqPath的对应的handler
	SetNotAllowedHandler(handler http.Handler) //reqPath对应的get,但是传来的确实post
}
