# DoH

A DNS-over-HTTP implementation written in Go


## TO DO

* implement CLI arguments support
** support verbose mode
** support profile
* implement redis or memcache as optional backend cache
* implement configuration support for config file or via CLI arguments
* implement DNS response parser
* implement freshness indicatore for cache-control: max-age
* optionally: implement DNS request message validator
* support basic statistics (i.e. for grafana)
* dockerize this app (with compose)