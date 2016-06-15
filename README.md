# SecurityCenter CLI *(written in Go)*

---
## Summary
This is a simple command line application to use various tasks capable in [Tenable Network Security](https://www.tenable.com)'s [Security Center](https://www.tenable.com/products/securitycenter) (>= v5.0) and manipulate data within SecurityCenter. This should make automating various tasks in SecurityCenter (such as automated backups, syncing assets from a local database or file into SC, auditing user acces, etc) much easier.

Built-in is also an interactive menu useful for exporting and importing records (Assets, Users, Groups, Scan Results). This menu, and mostly this entire command line application, was built based on requirements needed by West Virginia University's Information Security Services Office. I figured other organizations may be in need of the same easy-to-use tools so here they are open sourced.

Supported are Linux, Mac OS, and Windows environments via the `go build` tool.

## Third-Party Technologies

This was built using the following technologies:
* SecurityCenter - A vulnerability management and analytics solution by Tenable Network Security.
* [Go](https://golang.org) ( > 1.5)
    * [Bolt](https://github.com/boltdb/bolt) - "A fast key/value store inspired by [Howard Chu's LMDB project](https://symas.com/products/lightning-memory-mapped-database/)."
    * [cli](https://github.com/urfave/cli) - A library to help make building command line applications in Go easier.

## Documentation [![GoDoc](https://godoc.org/github.com/austindizzy/securitycenter-cli?status.svg)](https://godoc.org/github.com/austindizzy/securitycenter-cli)

This project contains various packages which are able to be imported into your Go application which may speed things up is you're also working with SecurityCenter's API, collection authentication interactively, or creating interactive submenus. Go documentation can be viewed on [GoDoc](https://godoc.org/github.com/austindizzy/securitycenter-cli).

Running `./securitycenter-cli help` will output some help documentation for using the command line application.

````
NAME:
   securitycenter-cli - a trusty cli for your trusty nvs

USAGE:
   securitycenter-cli [global options] command [command options] [arguments.
..]

VERSION:
   0.1a

AUTHOR(S):
   Austin Siford <Austin.Siford@mail.wvu.edu>

COMMANDS:
     export, x  export objects from SecurityCenter to a flat file
     import, i  import objects from a flat file to SecurityCenter
     test, c    test auth token for validity
     menu, m    start interactive menu
     auth, c    get/set auth tokens

GLOBAL OPTIONS:
   --host value             Tenable Nessus SecurityCenter API host [%TNS_HOST%]
   --token value, -t value  Auth token for SecurityCenter. [%TNS_TOKEN%]
   --session value          Auth session for SecurityCenter [%TNS_SESSION%]
   --debug                  Enable verbose logging.
   --help, -h               show help
   --version, -v            print the version
````

Specialized help documentation for each command can also be found by running `./securitycenter-cli [command] help`, for instance with `./securitycenter-cli export --help`

````
NAME:
   securitycenter-cli export - export objects from SecurityCenter to a flat
file

USAGE:
   securitycenter-cli export [command options] [data type to export]

OPTIONS:
   --fields value  fields to export
   --filter value  filter exported records
   --output value  optional file output
````
