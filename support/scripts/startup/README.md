## Timber Agent Example Startup Scripts

This directory contains example startup scripts for the Timber Agent on various
operating systems. For the most part, they are usable as-is if you follow
standard installation instructions when using a binary release package. (If
you're using an install package, the necessary script should already be
installed for you.)

Which example script you start with depends on the type of init system your
operating system uses. 

### SysV EPEL (Extra Packages for Enterprise Linux)

Despite the name, the EPEL script is what you want for Fedora systems prior to
version 15 (and all distros based on it).

This is also the script to use if you are on Amazon Linux.

### SysV LSB (Linux Standard Base)

The LSB script should be usable with init systems that are LSB-compliant. This
is the script you want if you use Debian Wheezy.
