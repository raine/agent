# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.0] - 2018-05-22

### Added

  - A new sub-command has been added, `capture-kube`, for collecting and
    shipping Kubernetes application logs.

## [0.7.0] - 2018-05-10

### Changed

  - CircleCI now uses Docker daemon 17.05.0-ce, the minimum version supporting multi-stage builds
  - When running `capture-files`, tailed file state is now stored in a single file at a configurable path

### Fixed

  - Fixed tests that would always fail when run on an AWS instance
  - Disable cgo when building the agent to avoid dynamically linking to C libs

## [0.6.2] - 2018-05-04

### Fixed

  - Fixed docker container failing to boot
  - Log lines larger than the max payload size (1MB) now are dropped as these would always receive an error code from the API.

## [0.6.1] - 2018-01-12

### Added

  - Releases will now build executables for Linux running on a 32-bit and 64-bit
    ARM processor

## [0.6.0] - 2017-12-29

### Changed

  - The "User-Agent" header now reports "timber-agent" along with the version
    number
  - Moved construction of the authorization header string out of a tight loop;
    this improves memory allocation by such an infinitesimal amount that it
    may be of no consequence

### Fixed

  - Fixed a disastrous memory leak that was occuring during the request cycle
  - Fixed metadata JSON encoding containing empty JSON objects which the service
    treated as invalid

## [0.5.4] - 2017-09-10
### Changed

  - The log data buffer size was reduced from 2mb to 990kb to adhere to Timber API limits.
  - The log data buffer flush interval was reduced from 10 seconds to 3 seconds to provide for more frequent delivery of log data.

## [0.5.3] - 2017-09-10
### Changed

  - All globbing special characters are now supported: *?[]

### Fixed

  - Resolves an issue where only the last file specified was being tailed.

## [0.5.2] - 2017-09-09
### Changed

  - Improved logging throughout the agent with improved activity reporting.
  - Only file paths with a * character will enable globbing. This makes file
    tailing stricter, treating paths without * as direct file paths.

## [0.5.1] - 2017-09-07
### Fixed

  - Elastic Beanstalk configuration script works again; a bug was introduced in
  0.5.0 that prevented it from being used properly

## [0.5.0] - 2017-09-07
### Added

  - Added support for globbing in file paths. Periodic checks ensure new files
  are recognized as well.

### Fixed

  - Resolves an issue where only the last file in the `[[files]]` array was
  being forwarded to Timber.io.

## [0.4.1] - 2017-08-29
### Added

  - Added a startup script for use with systemd based Linux distributions

## [0.4.0] - 2017-08-21
### Changed

  - Running `timber-agent` without a sub-command will now result in the help
    message being printed
  - The `--stdin` global switch has been replaced with the `capture-stdin`
    sub-command.
  - To capture log files defined in a configuration file, use the `capture-files`
    sub-command. This was previously the default operation when `--stdin` was
    not specified.
  - The flag `--agent-log-file` has been renamed `--output-log-file`

### Fixed

  - Capturing log data over stdin no longer requires a config file. A config file
    can still be set if you want to use it to provide an API key, override the
    hostname, or disable EC2 metadata collection
  - The `--daemonize` switch can no longer be used with capturing stdin


## [0.3.0] - 2017-08-09
### Added

  - Added example startup scripts for SysV style init systems
  - Added an example configuration file at `support/config/timber.basic.toml`
  - Added an example of log rotation configuration to
    `support/scripts/logrotate.d`
  - Releases now include `support` files in archives
  - Releases now include a copy of the README
  - Releases now include a copy of the CHANGELOG
  - Releases now include a copy of the LICENSE
  - Allows a user to configure a default API key using the `default_api_key`
    configuration key. If a file definition does not have an API key, the
    default API key will be used if it is present.

## [0.2.1] - 2017-08-04
### Fixed

  - If the AWS EC2 metadata service is available but returns HTTP errors (e.g.,
    404), we treat it as an error rather than taking the body as metadata.

## [0.2.0] - 2017-08-03
### Changed

  - API Keys in the config file should now be specified using the key name
    `api_key` instead of `apiKey`.

## [0.1.4] - 2017-08-02
### Fixed

  - AWS EC2 metadata now sources data from the correct key names

## [0.1.3] - 2017-08-02
### Added

  - Additional logs are produced during metadata collection. This should be
    helpful for dignostic purposes.

## [0.1.2] - 2017-08-02
### Fixed

  - Fixes an issue with the version of the libc library used to compile binaries
    for Linux distribution. Pre-compiled binaries for Linux should now run
    properly.

## [0.1.1] - 2017-08-02
### Changed

  - In distribution archives, the root of the archive contains the folder
    `timber-agent` which then contains the `bin` folder. This should make
    unpacking easier
  - The example AWS Elasticbeanstalk configuration has been updated to point to
    the correct archive location for the 0.1.x release line.

## [0.1.0] - 2017-08-01
### Removed

  - Daemonization using the `--daemonization` flag is no longer possible on Darwin
    (macOS) or any other BSD variant. Daemonization is only permitted on Linux.
    This is due to incompatibilities between the daemonization libraries and the
    other operating systems.

## [0.0.2] - 2017-07-28
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

[Unreleased]: https://github.com/timberio/agent/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/timberio/agent/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/timberio/agent/compare/v0.6.2...v0.7.0
[0.6.2]: https://github.com/timberio/agent/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/timberio/agent/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/timberio/agent/compare/v0.5.4...v0.6.0
[0.5.4]: https://github.com/timberio/agent/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/timberio/agent/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/timberio/agent/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/timberio/agent/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/timberio/agent/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/timberio/agent/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/timberio/agent/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/timberio/agent/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/timberio/agent/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/timberio/agent/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/timberio/agent/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/timberio/agent/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/timberio/agent/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/timberio/agent/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/timberio/agent/compare/v0.0.2...v0.1.0
[0.0.2]: https://github.com/timberio/agent/compare/v0.0.1...v0.0.2
