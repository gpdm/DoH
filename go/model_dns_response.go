/*
 * API DoH
 *
 * This is a DNS-over-HTTP (DoH) resolver written in Go.
 *
 * API version: 0.1
 * Contact: dev@phunsites.net
 */

package api_doh

type DnsResponse struct {
	Dns string `json:"dns,omitempty"`
}
