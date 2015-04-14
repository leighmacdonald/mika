# Mika

A torrent tracker written in the Go programming language and using redis
as a backend database.


## Features

Makes cake

## Usage

### Requirements

If building a binary from source you will need a Go 1.4+ SDK installed.


### Compiling
    
    go get github.com/garyburd/redigo/redis
    go get github.com/jackpal/bencode-go
    go get github.com/labstack/echo
    go get github.com/thoas/stats
    go build -o mika
    ./mika

### Run-time options:

* `-config <config.json>` - Path to config file. Default is ./config.json
* `-procs` - Number of processor cores to use. The default is $numcores-1, but you may want
to lower this if there is other contentious services running too.


### Signals

* `SIGUSR2` - Reload config


