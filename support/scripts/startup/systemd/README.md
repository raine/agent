## Timber Agent systemd Startup Script

This directory contains an example script for starting the Timber Agent at
startup on systemd-based Linux distributions. This covers the majority of Linux
distributions today.

### Using the Script

Copy the file `timber-agent.service` to `/etc/systemd/system/`. Make sure it is
owned by `root:root`. Once that's done, you'll need to tell the systemd daemon
to reload the unit files:

```sh
systemctl daemon-reload
```

Once this is done, you'll need to enable the unit so that the agent is started
on every boot:

```sh
systemctl enable timber-agent
```

To start the agent running immediately, use the `start` command:

```sh
systemctl start timber-agent
```

You can check the status according to systemd using the `status` command:

```sh
systemctl status timber-agent
```

You may need to use `sudo` in the commands above if you do not have the
appropriate permissions.

### Customizing

If you need to deviate from the default file locations, you can modify the
`ExecStart` line to meet your needs.
