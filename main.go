package main

import (
	"flag"
	"io"
	"log"
	"net/http"

	"tailscale.com/tsnet"
)

var (
	nodeUrl  = flag.String("nodeUrl", "", "url+port (https://myNode.tailXXXX.ts.net) of the tailscale node to route traffic to")
	port     = flag.String("port", ":8080", "address to listen and route your traffic to")
	hostname = flag.String("hostname", "iotProxy", "hostname to serve")
)

func main() {
	flag.Parse()

	server := new(tsnet.Server)
	server.Hostname = *hostname
	defer server.Close()

	// let this program listen on the tailscale network
	tsNetListener, err := server.Listen("tcp", *port)
	if err != nil {
		log.Fatal(err)
	}
	defer tsNetListener.Close()

	// Create a new http client that uses the tailscale network to route traffic to the tailscale node
	tsHttpClient := server.HTTPClient()

	// Listen and serve on the local network
	http.ListenAndServe(*port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if *nodeUrl == "" {
			http.Error(w, "Error: nodeUrl is required", http.StatusInternalServerError)
			return
		}

		// Create a new request to the tailscale node
		proxyReq, err := http.NewRequest(r.Method, *nodeUrl, r.Body)
		if err != nil {
			http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
			return
		}

		// Copy the headers from the original request to the proxy request
		for name, values := range r.Header {
			proxyReq.Header[name] = values
		}

		// Send the request to the tailscale node
		resp, err := tsHttpClient.Transport.RoundTrip(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy the headers from the proxy response to the original response
		for name, values := range resp.Header {
			w.Header()[name] = values
		}

		// Copy the status code from the proxy response to the original response
		w.WriteHeader(resp.StatusCode)

		// Copy the body from the proxy response to the original response
		io.Copy(w, resp.Body)
	}))
}
