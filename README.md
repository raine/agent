# timber-agent

This is a simple daemon that forwards logs to the timber.io service.

## Quickstart

Create a file at `/etc/timber.toml` and specify the following options:

```toml
[[files]]
path = "/var/log/app.log"
apiKey = "mytimberapikey"
```

Then simply run the agent and it will tail the given file, forwarding its
contents to the timber service using the provided API key.

## Configuration

Run `timber-agent help` to see the available options:

```
NAME:
   timber-agent - forwards logs to timber.io

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   0.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value  config file to use (default: "/etc/timber.toml")
   --stdin                   read logs from stdin instead of tailing files
   --api-key value           timber API key to use when forwarding stdin [$TIMBER_API_KEY]
   --help, -h                show help
   --version, -v             print the version
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
