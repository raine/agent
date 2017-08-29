# Timber Agent

[![Built by Timber.io](https://res.cloudinary.com/timber/image/upload/v1503615886/built_by_timber_wide.png)](https://timber.io/?utm_source=github&utm_campaign=timberio%2Fagent)

[![GitHub release](https://img.shields.io/github/release/timberio/agent.svg)](https://github.com/timberio/agent/releases/latest) [![license](https://img.shields.io/github/license/timberio/agent.svg)](https://github.com/timberio/agent/blob/master/LICENSE) [![CircleCI](https://img.shields.io/circleci/project/github/timberio/agent.svg)](https://circleci.com/gh/timberio/agent/tree/master)

The Timber Agent is a cross-platform utility for capturing log data and
sending it to Timber.io. It can be configured to watch log files for new
data or accept data over standard input (STDIN).

## Installing the Agent

Instructions for installing the agent are dependent on your target system.

Pre-compiled 64-bit binaries are available in distribution archives for
Linux, macOS, FreeBSD, NetBSD, and OpenBSD from the [repository's releases
page](https://github.com/timberio/agent/releases). The distribution packages
also contain example configuration and startup scripts.

Unpacking the distribution archive will leave you with a `timber-agent`
directory that should be placed in a common location like `/opt`. (The
instructions below will assume you place it in `/opt`; if you place it somewhere
different, you will need to adjust the paths appropriately.)

The binary for the agent will be located at `/opt/timber-agent/bin/timber-agent`.
The only requirement to run the agent (see Usage below), is a configuration
file. The agent will look for a configuration file at `/etc/timber.toml` by
default. If you use a different location, specify it using the `--config` flag.

An example configuration file is included at
`/opt/timber-agent/support/config/timber.basic.toml`.

## Quickstart

Create a file at `/etc/timber.toml` and specify the following options:

```toml
default_api_key = "timberapikey"

[[files]]
path = "/var/log/app/ruby.log"

[[files]]
path = "/var/log/app/puma.log"

[[files]]
path = "/var/log/nginx/access.log"
api_key = "different-api-key" # send this file to a different Timber application
```

Now, you can run the agent using `timber-agent capture-files`. The agent will
start in the foreground and begin capturing any new data written to the
specified files.

The agent will pick up the hostname of your server by default, but you can
explicitly set the hostname you want it to use with your logs by providing
a `hostname` key at the top of the file:

```toml
hostname = "worker-a.us-east-1.example.com"
```

## Usage
Run `timber-agent help` to see the available options:

```
NAME:
   timber-agent - forwards logs to timber.io

USAGE:
   timber-agent [global options] command [command options] [arguments...]

VERSION:
   0.4.1

COMMANDS:
     capture-stdin  Captures log data sent over STDIN and forwards to Timber's log collection endpoint
     capture-files  
     help, h        Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

### capture-stdin

```
NAME:
   timber-agent capture-stdin - Captures log data sent over STDIN and forwards to Timber's log collection endpoint

USAGE:
   timber-agent capture-stdin [command options] [arguments...]

OPTIONS:
   --api-key value           timber API key to use when capturing stdin [$TIMBER_API_KEY]
   --config value, -c value  config file to use (default: "/etc/timber.toml")
   --output-log-file FILE    the agent will write its own logs to FILE (will use STDOUT if not provided)
   --pidfile FILE            will store the pid in FILE when set

```

### capture-files

```
NAME:
   timber-agent capture-files -

USAGE:
   timber-agent capture-files [command options] [arguments...]

DESCRIPTION:
   Captures log data from files declared in configuration and forwards to Timber's log collection endpoint

OPTIONS:
   --config value, -c value  config file to use (default: "/etc/timber.toml")
   --daemonize               starts an instance of agent as a daemon (only available on Linux; see documentation)
   --output-log-file FILE    the agent will write its own logs to FILE (will use STDOUT if not provided)
   --pidfile FILE            will store the pid in FILE when set

```

## Contributing

This project uses [Dep](https://github.com/golang/dep) as the dependency manager
and all vendorized dependencies are committed into version control. If you make
a change that includes a new dependency, please make sure to add it to the
dependency manager properly. You can do this by editing the `Gopkg.toml` file in
the root of the project ([format
documentation](https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md)).
After editing the file, run `dep ensure` to update the `vendor` folder.


## LICENSE

The original parts of this software as developed by Timber Technologies, Inc. as
well as contributors are licensed under the Internet Systems Consortium (ISC)
License. This software is dependent upon third-party code which is
statically linked into the executable at compile time. This third-party code is
redistributed without modification and made available to all users  under the
terms of each project's original license within the `vendor` directory of the
project.

