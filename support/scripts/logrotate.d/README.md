## Timber Agent Logrotate Examples

The files in this directory give example configurations for setting up log
rotation on the Timber Agent's logs. The examples all expect the log file to be
located at the recommended location `/var/log/timber-agent.log`. They rotate the
file up to two times, never allowing it to grow past 100 kb in size.

The primary difference between the files is how they handle restarting the agent
after log rotation.

The agent will continue to write to the file after it is rotated which is why
a conditional restart is needed. This is also why if you wish to compress the
rotated file, you need to set `delaycompress`.

The `timber-agent-systemd` file will use `systemctl` to restart the agent.

The `timber-agent-sysv` file will use the `service` utility to restart the
agent.

Which version you need depends on which init system your operating system uses.
On most contemporary Linux distributions, you will use the `systemd` version of
the file.

To use this file, copy it into `/etc/logrotate.d/`.
