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
