# Timber Agent

[![Built by Timber.io](https://res.cloudinary.com/timber/image/upload/v1503615886/built_by_timber_wide.png)](https://timber.io/?utm_source=github&utm_campaign=timberio%2Fagent)

[![GitHub release](https://img.shields.io/github/release/timberio/agent.svg)](https://github.com/timberio/agent/releases/latest) [![license](https://img.shields.io/github/license/timberio/agent.svg)](https://github.com/timberio/agent/blob/master/LICENSE) [![CircleCI](https://img.shields.io/circleci/project/github/timberio/agent.svg)](https://circleci.com/gh/timberio/agent/tree/master)

The Timber Agent is a cross-platform natively-compiled utility for capturing log data
(file & STDIN) and sending it to Timber.io. It is designed to be light weight, highly efficient,
and reliable without the need for dependencies.

1. [**Installation**](#installation)
2. [**Usage**](#usage)
3. [**Configuration**](#configuration)
4. [**Contributing**](#contributing)


## Installation

1. Download the archive for your architecture:

    ```shell
    curl -LO {{choose-url-below}}
    ```

    * [Darwin AMD64 latest](https://packages.timber.io/agent/0.x.x/darwin-amd64/timber-agent-0.x.x-darwin-amd64.tar.gz)
    * [FreeBSD AMD64 latest](https://packages.timber.io/agent/0.x.x/freebsd-amd64/timber-agent-0.x.x-freebsd-amd64.tar.gz)
    * [Linux AMD64 latest](https://packages.timber.io/agent/0.x.x/linux-amd64/timber-agent-0.x.x-linux-amd64.tar.gz)
    * [Netbsd AMD64 latest](https://packages.timber.io/agent/0.x.x/netbsd-amd64/timber-agent-0.x.x-netbsd-amd64.tar.gz)
    * [Openbsd AMD64 latest](https://packages.timber.io/agent/0.x.x/openbsd-amd64/timber-agent-0.x.x-openbsd-amd64.tar.gz)

    All releases can be found [here](https://github.com/timberio/agent/releases). Special URLs that point to the current releases can be found [here](https://timber.io/docs/platforms/other/agent/versioning).

2. Unpack the archive to a common location like `/opt`:

    ```shell
    tar -xzf timber-agent-0.x.x-darwin-amd64.tar.gz -C /opt
    ```

    The agent will be located at `/opt/timber-agent/bin/timber-agent`.

3. Move the `timber.toml` file to `/etc`:

    ```shell
    cp /opt/timber-agent/support/config/timber.basic.toml /etc/timber.toml
    ```

4. In `/etc/timber.toml` replace `MY_TIMBER_API_KEY` with your API key. [*Don't have a key?*](https://timber.io/docs/app/applications/obtaining-api-key)

    ```shell
    sed -i 's/MY_TIMBER_API_KEY/{{my-timber-api-key}}/g' /etc/timber.toml
    ```

    *Be sure to replace `{{my-timber-api-key}}` in the command above with your _actual_ API key.*

5. In `/etc/timber.toml` update the `[[files]]` entries to forward your chosen files.

6. Start the `timber-agent`:

    ```shell
    /opt/timber-agent/bin/timber-agent capture-files
    ```

    Checkout the [usage section](#usage) as well as the [startup scripts directory](https://github.com/timberio/agent/tree/master/support/scripts/startup) to assist with starting and stopping the agent.


#### How it works

The agent will start in the foreground and begin capturing any new data written to
the specified files.

The agent will also pick up the hostname of your server by default, but you can
explicitly set the hostname you want it to use with your logs by providing
a `hostname` key at the top of the file:

```toml
hostname = "worker-a.us-east-1.example.com"
```

See [configuration](#configuration) below for more details.


## Usage

Run `timber-agent help` to see the available options:

```
NAME:
   timber-agent - forwards logs to timber.io

USAGE:
   timber-agent [global options] command [command options] [arguments...]

VERSION:
   0.6.1

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
   --config value, -c value  config file to use, for available options see https://timber.io/docs/platforms/other/agent/configuration-file (default: "/etc/timber.toml")
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
   --config value, -c value  config file to use, for available options see https://timber.io/docs/platforms/other/agent/configuration-file (default: "/etc/timber.toml")
   --daemonize               starts an instance of agent as a daemon (only available on Linux; see documentation)
   --output-log-file FILE    the agent will write its own logs to FILE (will use STDOUT if not provided)
   --pidfile FILE            will store the pid in FILE when set
   --statefile value         File path for storing global state, defaults to sane path based on OS
```


## Configuration

Outside of the usage options specified above, the agent takes a config file.
The default path for this config file is `/etc/timber.toml`. Avilable options
can be found in [our docs](https://timber.io/docs/platforms/other/agent/configuration-file).


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
