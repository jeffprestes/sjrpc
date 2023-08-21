# Save JSON-RPC

**sjrpc** or **Save JSON-RPC** is a Reverse Proxy focused on to reduce remote Ethereum/Web3 JSON-RPC calls to thrid-party nodes.

## How it works

Using an embedded and local [BadgerDB](https://github.com/dgraph-io/badger) database, **sjrpc** hashes the Request,
uses the request hash as Key, perform the remote JSON-RPC call and saves the remote response locally. At next call it gets the content from the local database.

## Installation

### Download

- [Mac Silicon]("downloads/sjrpc-v0.1.0-mac-silicon")
- [Mac Intel]("downloads/sjrpc-v0.1.0-mac-intel")
- [Linux]("downloads/sjrpc-v0.1.0-linux-amd64")
- [Windows]("downloads/sjrpc-v0.1.0-windows-amd64.exe")

## Installation from sources

### Install Go Language

Visit [Golang Download and Install page](https://go.dev/doc/install) and follow the instructions.

### Clone this repository

### Compile it

In case you use Mac or Linux, install `make` if you don't have it already, then run:

```bash
make build
```

In case you use Windows, run:

```powershell
go build -o bin/sjrpc.exe cmd/main.go
```

## Usage

### Set your RPC Server URL using environment variables

Set `SJRPC_URL` with your remote RPC Server URL

Example:

```shell
export SJRPC_URL=https://mainnet.infura.io/v3/<YOUR INFURA API KEY>
```

### Run it

In case you use Mac or Linux run:

```bash
make run
```

In case you use Windows, run:

```powershell
bin/sjrpc.exe
```

It will start a localhost Web Server on port 8434. This port number is fixed to avoid the risk you mess it with standard dev ETH node port: 8545


### Call it

Configure your Foundry, Truffle, Go, Hardhat or any Web3 application to use this RPC server: `http://localhost:8434`.

### Clean Up

When you need to call another Blockchain network your cache gets outdate and you need to clean it up. To do so you need to call the `cleanup` endpoint:

http://localhost:8434/cleanup

## Perfomance hint

It runs better in 64-bit architect processors, such M1/M2 Apple chips, or Intel i7. The reason is it uses Blake2b 512 bits.

## TODOs

- Manage block creation time to allow to cache requests that uses `latest` param meanwhile a new block is not generated.
