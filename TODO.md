## TO DO

Here's the list of still missing things to be done, in order of priority.

* parser/normalizer for dns.resolvers config properties
* Internal connectivity poller for upstream and sidecar services, to gracefully handle outages on DNS resolvers, InfluxDB and Redis
* Relay internal log data to remote log server, i.e. syslog, or other log collector facilities.
* Telemetry for DNS backends to include response time per queried DNS server
* Rework DNS backend support: Support DNS-over-TLS as well
* Implement a Docker compose file
* Implement a health check mechanism
