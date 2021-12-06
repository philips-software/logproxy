# Change Log
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## v1.7.0

- Support IAM Service Identities

## v1.6.0

- Support true passthrough of LogEvent

  **NOTE**: `LogData.Message` field must be base64 encoded!

## v1.5.2

- Feature: support LogEvent passthrough (@rostbel)

## v1.5.1

- Update structured log with new fields

## v1.5.0
- Add traceId and spanId fields

## v1.4.3
- Build fixes

## v1.4.2
- Inital OpenTracing support

## v1.4.1
- Fix tagged build

## v1.4.0
- Maintenance release

## v1.3.3
- Maintenance release

## v1.3.2
- Add spans for APM

## v1.3.1
- Elastic APM support

## v1.3.0
- Bugfix: Fix dropping of inst field when using DHPLog messages

## v1.2.2
- Filter only mode

You may choose to operate Logproxy in Filter only mode. It will listen
for messages on the logdrain endpoints, run these through any active
filter plugins and then discard instead of delivering them to HSDP logging.
This is useful if you are using plugins for real-time processing only.
To enable filter only mode set LOGPROXY_DELIVERY to none

...
env:
  LOGPROXY_DELIVERY: none
...

##  v1.2.1
- Minor fixes

## v1.2.0

- Plugin support
- Go channel queue support

## v1.1.1

- Fix procID field encoding
- Update dependencies
- Improve code coverage

## v1.1.0

- Encode invalid chars in messages
- Support for December 2019 HSDP logging release (custom fields) 
- IronIO logdrain support

## v1.0.0

- Initial OSS release
- Extract methods to [go-hsdp-api](https://github.com/philips-software/go-hsdp-api)
