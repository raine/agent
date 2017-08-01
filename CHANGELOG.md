# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Removed
- Daemonization using the `--daemonization` flag is no longer possible on Darwin
  (macOS) or any other BSD variant. Daemonization is only permitted on Linux.
  This is due to incompatibilities between the daemonization libraries and the
  other operating systems.

## 0.0.2 - 2017-07-28
### Added
- Automatically collects hostname from the operating system and sends it as metadata
  to the server; allows the value to be overridden by setting the `hostname` parameter
  in the configuration file
- Automatically collects EC2 instance metadata if available. This can be disabled by
  setting the `disable_ec2_metadata` parameter in the configuration file to `true`

## 0.0.1 - 2017-07-13
### Added
- Ability to collect logs from files and upload them to the Timber Hosted service by
  specifying paths in a configuration file
- Files specified via a configuration file will be kept track of; if the agent
  stops at any point, it will make an effort to resume tailing at the point it
  stop
- Ability to upload logs by streaming them over STDIN

[Unreleased]: https://github.com/timberio/agent/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/timberio/agent/compare/v0.0.1...v0.0.2


