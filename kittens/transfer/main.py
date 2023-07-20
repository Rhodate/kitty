#!/usr/bin/env python
# License: GPLv3 Copyright: 2021, Kovid Goyal <kovid at kovidgoyal.net>


import sys
from typing import List

usage = 'source_files_or_directories destination_path'
help_text = 'Transfer files over the TTY device'


def option_text() -> str:
    return '''\
--direction -d
default=send
choices=send,receive
Whether to send or receive files.


--mode -m
default=normal
choices=normal,mirror
How to interpret command line arguments. In :code:`mirror` mode all arguments
are assumed to be files on the sending computer and they are mirrored onto the
receiving computer. In :code:`normal` mode the last argument is assumed to be a
destination path on the receiving computer.


--compress
default=auto
choices=auto,never,always
Whether to compress data being sent. By default compression is enabled based on the
type of file being sent. For files recognized as being already compressed, compression
is turned off as it just wastes CPU cycles.


--permissions-bypass -p
The password to use to skip the transfer confirmation popup in kitty. Must match
the password set for the :opt:`file_transfer_confirmation_bypass` option in
:file:`kitty.conf`. Note that leading and trailing whitespace is removed from
the password. A password starting with :code:`.`, :code:`/` or :code:`~`
characters is assumed to be a file name to read the password from. A value of
:code:`-` means read the password from STDIN. A password that is purely a number
less than 256 is assumed to be the number of a file descriptor from which to
read the actual password.


--confirm-paths -c
type=bool-set
Before actually transferring files, show a mapping of local file names to remote
file names and ask for confirmation.


--transmit-deltas -x
type=bool-set
If a file on the receiving side already exists, use the rsync algorithm to
update it to match the file on the sending side, potentially saving lots of
bandwidth and also automatically resuming partial transfers. Note that this will
actually degrade performance on fast links with small files, so use with care.
'''


def main(args: List[str]) -> None:
    raise SystemExit('This should be run as kitten transfer')


if __name__ == '__main__':
    main(sys.argv)
elif __name__ == '__doc__':
    cd = sys.cli_docs  # type: ignore
    cd['usage'] = usage
    cd['options'] = option_text
    cd['help_text'] = help_text
    cd['short_desc'] = help_text
