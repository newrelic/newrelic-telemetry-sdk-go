# ChangeLog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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


[Unreleased]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/newrelic/newrelic-telemetry-sdk-go/releases/tag/v0.1.0