# ChangeLog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## Unreleased

* Adds support for creating event groups for the request factory

## [0.7.0] - 2021-04-02

### Breaking Changes ⚠️ 
* BuildRequest on Request factories now have new interfaces to reflect the outline of the payload. Helpers for common blocks and groups are provided.

### Performance Improvements 🚀 
* Buffer allocations are now minimized within the request factory via internal buffer caching and re-use.

### Bug fixes 🧯
* Large payloads were not automatically split when using the harvester. Payloads are now split to reduce payload size when required.

## [0.6.0] - 2021-03-17
### Added
- Adds support for sending log data to New Relic.
- Add `ClientOption` for specifying gzip compression level. Use:
  `WithGzipCompressionLevel`.

## [0.5.2] - 2021-03-02
### Added
- Adds a RequestFactory API that can be used for managing the telemetry data
requests if you need more fine-grained control than the Harvester API supports.
Only Span data is currently supported by this API.
### Fixed
- Fix performance issue caused by the gzip writer being reallocated for each
request - it's now reused between requests. 

## [0.5.1] - 2020-12-16
- Fixed bug that resulted in payload size remaining slightly too large after
splitting it into more manageable chunks. (#39)

## [0.5.0] - 2020-11-19
### Added
- Implemented preliminary OpenTelemetry span support. (#31)

## [0.4.0] - 2020-08-04
### Fixed
- Fixed bug in request retrying that resulted in the a zero length request
body and manifested as an error mismatch in body length and Content-Length
header. (#17)

## [0.3.0] - 2020-06-12
### Added
- Added `ConfigSpansURLOverride` to facilitate setting the Trace Observer URL
for Infinite Tracing on the New Relic Edge. (#15)

## [0.2.0] - 2019-12-26
### Fixed
- The SDK will now check metrics for infinity and NaN.  Metrics with invalid
values will be rejected, and will result in an error logged. (#3)

### Added
- Added `Config.Product` and `Config.ProductVersion` fields which are
used to the `User-Agent` header if set. (#2)

## [0.1.0]
First release!

[Unreleased]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.5.2...v0.6.0
[0.5.2]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/releases/tag/v0.1.0
