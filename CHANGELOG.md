## ChangeLog

## 0.4.0

* Fixed bug in request retrying that resulted in the a zero length request
body and manifested as an error mismatch in body length and Content-Length
header.

## 0.3.0

* Added `ConfigSpansURLOverride` to facilitate setting the Trace Observer URL
  for Infinite Tracing on the New Relic Edge.

## 0.2.0

* The SDK will now check metrics for infinity and NaN.  Metrics with invalid
values will be rejected, and will result in an error logged.

* Added `Config.Product` and `Config.ProductVersion` fields which are
used to the `User-Agent` header if set.

## 0.1.0

First release!
