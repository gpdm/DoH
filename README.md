# DoH Server

This is a "DNS over HTTP" (DoH) server implementation written in Go.

The implementation follows RFC8484[https://tools.ietf.org/html/rfc8484], and provides several key features:

* support for both POST and GET queries over HTTP/2 and TLS
* supports one or more backend DNS servers
* optional support to send telemetry information to InfluxDB
* configuration support through config files and environment vars
* supports an optional HTTP-only (read: unencrypted) variant, primarily intended for debugging and development purposes
* intended to be leightweight and compact
* support to run from Docker comes for free

What this DoH implementation is not:

* This is *not* a DNS server itself, and never will be. It's intended to proxy DoH requests against existing DNS backend servers.
* This is *not* an attempt in breaking privacy. Read more on the development motivation below.

Known Limitations:

* Only traditional DNS servers responding on UDP:53 are supported for now
* Incoming request packets are not validated, thus relayed 1:1 to the DNS backend server(s)


## Motivation

First, I wanted to understand, how exactly DoH works, and what pitfalls it brings.
From this, the idea spawned to actually polish this daemon, so it could be easily "plugged" into any existing network infrastructure
to run a local DoH service yourself.



## Running the server

### Running with Docker

it's the primarily intended mode of operations to run the DoH daemon from Docker.

```
docker run .....
```

### CRIY (Compile and Run It Yourself)

To compile your self, do this in the source directory:

```
go build
```


To run it, here's a short excerpt on the CLI args:

```
Usage of ./DoH:
  -configfile string
        config file (optional)
  -debug
        debug mode
  -verbose
        verbose mode
```

As you see, there's not too many options. A sample config file is provided beneath `./conf/DoH.toml.sample`, I'll cover that further below.


### Configuration Directives

This section covers available configuration directives.
They can either be set from environment variables (useful for Docker), or from the configuration file.






## A Personal Opinion on DoH

To make it absolutely clear: I endorse the argument of added privacy enforced by using DoH over traditional, unencrypted DNS transport.
However, I also do have my strongs concerns about certain things.

If you don't care, simply skip this section on my personal opinion ;-)


### Browsers support DoH using Centralized Providers

Both Firefox and Chrome gained DoH support and are ready to send DNS queries over to either CloudFlare or Google.
Throwing the queries over to centralized facilities goes against the principles and building foundations of the Internet,
which had a decentral setup in mind.

And what serves the purpose of encrypted transmission, if data is collected at large scale with two huge global players?

It's easy to do data analysis by just looking at the logs, profiles could be built by just looking at the source IPs and
correlating with the DNS questions.

Yes, practically everybody running a DNS server can do this. And yes, even my implementation provides a debugging mode,
which could be abused for doing such nasty things.

The point is: DoH should run locally, and be connected to your own local DNS *recursive resolver* (not to mistake this with a *forwarding-only resolver*).


### DoH in Browsers bypass the local DNS resolver

DoH in the current form is an overlay transport mechanism, which is implemented in the Browsers, and bypasses your locally
configured DNS resolvers of the Operating System.

This not only that the queries pass-by your local resolvers, it also takes away some control from the local network administrator.

Depending on the settings (.e.g. Firefox, see TRR[https://wiki.mozilla.org/Trusted_Recursive_Resolver]), the browser
can be taught to either ignore the local resolver entirely, or only take it into account *after* the DoH recursion.

Going for privacy, this is great, as it bypasses any locally enforced DNS policy (.e.g to blocking of unwanted websites) at once.
At the same time, this is bad, because it bypasses any locally enforced DNS Policy at once.

I consider this sort of a double-edged sword.


### Freedom of Information Availability vs. the Law

In some countries, any institution may be obliged by law to enforce certain access and content to be blocked.
Sometimes, an institution (let's say, a school) may even willingly decide to enforce certain blocking policies, say for ethical reasons.

Some people consider this "censorship". I am against censorship, and everybody is free to ask my about my opinion
on Switzerlands law to block out foreign online casions. Worst. Thing. Ever.

However: There is unarguably some content on the web, which is definitely not suitable to pupils of a certain age.
So a school IMO must have the authority to enforce certain blocking rules, DoH takes this away entirely.

In addition: As long as the gorvernors of any organization can be potentially held liable for not blocking certain content,
DoH is simply not the way to go.


### Compatibility and Other Issues

* DoH may cause problems with DNS views on certain setup.
* DoH may and propably will break geo-based DNS load balancing (i.e. Akamai uses such mechanisms)
* DoH cannot entirely replace the traditional operating system resolver, i.e. when encountering RFC1918 IPs in a response, default is to fallback to the OS resolver.
* It's not natively supported (yet) by the operating systems
* The implementation has to compete against DNS over TLS (DoT), which itself has the same issue of non wide-spread support
* DoH adds extra overhead, i.e. every application needs to implement an extra layer to support both traditional and DoH resolvers. While understandable for the reason *why*, it feels unnatural and not the right way for me. This should be in the OS resolver library. Period.
* I'm still looking into it, but the HTTP Caching Topic may cause headaches as well
* Did I mention, it takes away the authority from the local network admin?


### What I Like about DoH

* The overlay protocol is very lightweight. It was a good choice to not go for JSON-based encoding, but use the DNS wire format.
* It was good to go for HTTP/2 right away, which implies TLS as well
* Clients implementations are enforced to only support TLS
* Given the fact that no OS native implementation yet exists, pushing it to the browsers is understandable in order to give the protocol a push towards getting widely deployed. I do hope however that OS implementers will eventually add support for both DoH and/or DoT.



## TO DO

Here's the list of still missing things to be done, in order of priority:

* Implement optional query caching using memcache or Redis
* implement freshness indicator for "cache-control: mag-age" as per RFC8484
* Telemetry for DNS response time per queried DNS server
* Rework DNS backend support: Support both DNS-over-TLS and DNS-over-HTTP resolvers as well
* Relay internal log data to remote Syslog server
* More code cleanups
