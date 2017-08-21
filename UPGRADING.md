# Upgrading Instruction

## Upgrade to 0.4

The 0.4 release introduces the use of sub-commands in the agent. You must now
specify either `capture-stdin` or `capture-files` as the sub-command when
calling the agent.

For example, if you have the agent set up like this:

```
timber-agent --stdin --api-key "12345"
```

you must now use:

```
timber-agent capture-stdin --api-key "12345"
```

Likewise, if you have the agent set up to run as a daemon like this:

```
timber-agent --daemonize --pidfile $PIDFILE
```

you must now use:

```
timber-agent capture-files --daemonize --pidfile $PIDFILE
```

You must also change use of the flag `--agent-log-file` to `--output-log-file`
