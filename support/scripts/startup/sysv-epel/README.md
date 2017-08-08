## Timber Agent EPEL Startup Script

This directory contains an example script for starting the Timber Agent with
Linux distribtutions that expect [EPEL style SysV init
scripts](https://fedoraproject.org/wiki/EPEL:SysVInitScripts?rd=Packaging:SysVInitScript).
This is the case for CentOS 6 and below as well as Fedora 15 and below. It is
also the case for Amazon Linux.


### Using the Script

Copy the file `timber-agent` into `/etc/rc.d/init.d/`. Make sure it is owned by
`root:root` with executable permissions for the owner.

```sh
cp /opt/timber-agent/support/scripts/startup/sysv-epel/timber-agent
/etc/rc.d/init.d/
chown root:root /etc/rc.d/init.d/timber-agent
chmod 755 /etc/rc.d/init.d/timber-agent
```

Once that's done, you can activate the script using `chkconfig`:

```sh
chkconfig --add timber-agent
```

The Timber Agent will now be started automatically during run levels 3, 4,
and 5. To start it immediately (and not wait for a reboot), use the `service`
tool:

```sh
service timber-agent start
```

If you need to check the status of the agent at any point in time, you can use
the `status` argument:

```sh
service timber-agent status
```

You may need to use `sudo` in the commands above if you do not have the
appropriate permissions.

### Customizing

The following default values are assigned in the script

```sh
config_file=/etc/timber.toml
log_file=/var/log/timber-agent.log
pid_file=/var/run/timber-agent.pid
```

If you would like to override these values, create a file at
`/etc/sysconfig/timber-agent` that redefines the variables.
