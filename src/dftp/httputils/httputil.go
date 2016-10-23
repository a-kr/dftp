package httputils

import (
	"dftp/utils"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

func HandleFunc(mux *http.ServeMux, url string, handler func(http.ResponseWriter, *http.Request)) {
	handler = panicCatcherMiddleware(handler)

	mux.HandleFunc(url, handler)
}

func panicCatcherMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				stack := utils.GetTraceback()
				errinfo := fmt.Sprintf("ERROR: PANIC: %s\n%s", x, stack)
				log.Printf("%s", errinfo)
				http.Error(rw, errinfo, 500)
			}
		}()
		next(rw, r)
	}
}

func MakeTimeoutingHttpClient(timeout time.Duration) *http.Client {
	timeoutFn := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	transport := http.Transport{
		Dial: timeoutFn,
		ResponseHeaderTimeout: timeout,
	}

	client := http.Client{
		Transport: &transport,
	}
	return &client
}
