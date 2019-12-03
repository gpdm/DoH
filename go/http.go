package dohservice

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// See https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
func newHTTPClient() *http.Client {
	c := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second, // This is the default.
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}
	return c
}
