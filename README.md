Go Bulk DNS Resolver
====================

About
-----
Lightning-fast high-performance bulk DNS resolution tool written in [Go](http://golang.org/) based on [miekg/dns](https://github.com/miekg/dns).

Requirements
------------
    - OSX: `brew update && brew install unbound`

    - Linux: `sudo apt-get update && sudo apt-get install unbound libunbound-dev

Building
--------
    ./build.sh

Examples
--------
The input is a newline-delimited list of domain names or URLs.  The output will be of the form `<domain> <ip1> <ip2> .. <ipN>`.

    echo 'google.com' | ./bulkdns
	google.com 74.125.239.34 74.125.239.46 74.125.239.38 74.125.239.33 74.125.239.36 74.125.239.35 74.125.239.40 74.125.239.39 74.125.239.32 74.125.239.37 74.125.239.41

Options
-------
`-p` will keep the output prefix exactly like the input prefix.  By default, the behavior is to prefix the output with only the domain name.


LICENSE
-------
go-bulk-dns-resolver

Copyright (C) 2014 - ThreatStream

This program free software; you can redistribute it and/or modify it under the terms of the GNU Lesser General Public License as published by the Free Software Foundation; either version 2.1 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License along with this program; if not, write to the Free Software Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
