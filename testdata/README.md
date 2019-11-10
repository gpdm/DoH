# DoH Test Files

This is sample test files used for automatic testing of the DoH implementation.

All files follow the following convention:

```
<REQUEST-TYPE>_<FQDN>.<PAYLOAD-TYPE>
```

where as:

* REQUEST-TYPE indicates the type of DNS request, i.e. `A`, `AAAA`, `NS`, etc
* FQDN indicates the FQDN to be queried
* PAYLOAD-TYPE indicates the payload encoding, as in
  * `.bd64` is Base64 (RFC4648 URL-encoded style without padding, as used for GET requests)
  * `.bin` is in plain binary (wire format) for POST requests
 
