/*
 * API DoH
 *
 * This is a DNS-over-HTTP (DoH) resolver written in Go.
 *
 * API version: 0.1
 * Contact: dev@phunsites.net
 */

package dohservice

type DNSResponse struct {
	DNS string `json:"dns,omitempty"`
}
