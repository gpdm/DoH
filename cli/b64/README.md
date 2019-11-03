# b64

A simple CLI utility for converting captured wire-format DNS requests into Base64

## Synopsis

I wrote this since the `base64` does not perform URL format encoding.

This format as documented in https://tools.ietf.org/html/rfc4648#section-5
is slightly different from the regular base64 encoding, and is mandatory
to use when using the DoH GET method, as noted in https://tools.ietf.org/html/rfc8484#section-4.1.1.

### Encoding Comparison

Here's an example on the subtle difference for the two encodings.

Standard Base64 encoding:

```
Fb4BIAABAAAAAAABAAACAAEAACkQAAAAAAAAAA==
```

URL-friendly encoding, without padding:

```
Fb4BIAABAAAAAAABAAACAAEAACkQAAAAAAAAAA
```

While not so obvious, here's another example, shamelessly stolen from RFC8484:

Standard Base64:

```
AAABAAABAAAAAAAAAWE+NjJjaGFyYWN0ZXJsYWJlbC1tYWtlcy1iYXNlNjR1cmwtZGlzdGluY3QtZnJvbS1zdGFuZGFyZC1iYXNlNjQHZXhhbXBsZQNjb20AAAEA
```

URL-friendly encoding:

```
AAABAAABAAAAAAAAAWE-NjJjaGFyYWN0ZXJsYWJlbC1tYWtlcy1iYXNlNjR1cmwtZGlzdGluY3QtZnJvbS1zdGFuZGFyZC1iYXNlNjQHZXhhbXBsZQNjb20AAAEA
```


### Syntax

The utility does just take two parameters, as shown:

```
$ ./b64  -h
Usage of ./b64:
  -infile string
    	input file name (required)
  -outfile string
    	output file name (required)
```

The `b64` utility will try it's best to (very cheaply) detect the input format, and convert
from wire-format to base64, and vice-versa.

If you need a proper DNS packet, it's best to capture it using TCPDUMP and store the packet payload as file,
then convert it using `b64` to the base64 format.

Please note:
No validation is done on the binary wire-format input format, so basically anything can be thrown at it.
This utility is a quick&dirty helper to assist in the API development, not a fool-proof solid thing.