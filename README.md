# Subscan 

Subscan is forked from Subscan Essentials supported by web3 foundation, which provided substrate-based blockchain explorer include observer and HTTP API server

## Contents

- [Feature](#Feature)
- [QuickStart](#QuickStart)
  - [Requirement](#Requirement)
  - [Structure](docs/tree.md)
  - [Installation](#Install)
  - [Config](#Config)
  - [Usage](#Usage)
  - [Docker](#Docker)
  - [Test](#Test)
- [Resource](#Resource)

## Feature

1. Support Substrate network [custom](/custom_type.md) type registration 
2. Support index Block, Extrinsic, Event, log
3. More data can be indexed by custom [plugins](/plugins)
4. [Gen](https://github.com/itering/subscan-plugin/tree/master/tools) tool can automatically generate plugin templates
5. Built-in default HTTP API [DOC](/docs/index.md)


## QuickStart

### Requirement

* Linux / Mac OSX
* Git
* Golang 1.12.4+
* Redis 3.0.4+
* MySQL 5.6+
* Node 8.9.0+

### Install

```bash
./build.sh build

#### Feature Supported

- search block detail by block number or block hash
- search extrinsic detail by extrinsic index or extrinsic hash
- search runtime info by spec version
- plugin (blocks, events)


### Config

#### Init config file 

```bash
cp configs/redis.toml.example configs/redis.toml && cp configs/mysql.toml.example configs/mysql.toml && cp configs/http.toml.example configs/http.toml
```

#### Set

1. Redis  configs/redis.toml

> addr： redis host and port (default: 127.0.0.1:6379)

2. Mysql  configs/mysql.toml

> host: mysql host (default: 127.0.0.1)
> user: mysql user (default: root)
> pass: mysql user passwd (default: "")
> db:   mysql db name (default: "subscan")

3. Http   configs/http.toml

> addr: local http server port (default: 0.0.0.0:4399)


### Usage

- Start DB

**Make sure you have started redis and mysql**

- Substrate Daemon (example in substrate)
```bash
cd cmd
./subscan start substrate
```

- Substrate Plugins 
```bash
cd cmd
./subscan start plugins
```

- Substrate Repair  
```bash
cd cmd
./subscan start repair
```

- Api Server
```bash
cd cmd
./subscan
```

- Help 

```
NAME:
   SubScan - SubScan Backend Service, use -h get help

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   1.0

DESCRIPTION:
   SubScan Backend Service, substrate blockchain explorer

COMMANDS:
     start    Start one worker, E.g substrate
     stop     Stop one worker, E.g substrate
     install  Create database and create default conf file
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --conf value   (default: "./configs")
   --help, -h     show help
   --version, -v  print the version


```

### Docker

Use [docker-compose](https://docs.docker.com/compose/) can start projects quickly 

Create local network

```
docker network create app_net
```

Run mysql and redis container

```bash
docker-compose -f docker-compose.db.yml up  -d
```

Run subscan service

```bash
docker-compose build
docker-compose up -d
```

### Test


**default test mysql database is subscan_test. Please CREATE it or change configs/mysql.toml**

```bash
go test -v ./...
```
### deprecated Collecting Logs

When launch subscan with observer daemon, it would generate a log folder named log from rootPath include a substrate_log contains logging info 
```
./log/substrate_log
```
### Logs
Each observer daemon would generate log to stdout, so can collect log in normal way

## Resource
 
- [ITERING] https://github.com/itering
- [Darwinia] https://github.com/darwinia-network/darwinia