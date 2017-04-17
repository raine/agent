# timber-agent

This is a simple daemon that forwards logs to the timber.io service.

## Quickstart

Create a file at `/etc/timber.toml` and specify the following options:

```toml
file = "/var/log/app.log"
api-key = "mytimberapikey"
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
   --config value, -c value  location of the config file to read (default: "/etc/timber.toml")
   --stdin                   read logs from stdin instead of a file
   --file value              log file to forward
   --batch-period value      how often to flush logs to the server (default: 5s)
   --poll                    poll files instead of using inotify
   --api-key value           your timber API key
   --endpoint value          the endpoint to which to forward logs (default: "https://ingestion-staging.timber.io/frames")
   --help, -h                show help
   --version, -v             print the version
```

Each option (excluding `--config` and `--stdin`) can be specified either in the
TOML configuration file or on the command line, with the command line taking
precedence.
