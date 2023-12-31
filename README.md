# Save JSON-RPC

**sjrpc** or **Save JSON-RPC** is a Reverse Proxy focused on to reduce remote Ethereum/Web3 JSON-RPC calls to thrid-party nodes.

It is the NGINX for Ethereum JSON-RPC APIs

However, let us save your time, if you use:

```javascript
// Example of Javascript/Typescript static frontend getting Ethereum RPC connection via browser embedded connection, such as Metamask
if (window.ethereum) { 
  ...
}
```

This project is not for you. Period.

## How it works

Using an embedded and local [BadgerDB](https://github.com/dgraph-io/badger) database, **sjrpc** hashes the Request,
uses the request hash as Key, perform the remote JSON-RPC call and saves the remote response locally. At next call it gets the content from the local database.

## Security

As it keeps the cache data locally, your project does not face a risk to get tampered data. We do not recommend you expose it externally.

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

### Usage

Configure your Foundry, Truffle, Go, Hardhat or any Web3 application to use this RPC server: `http://localhost:8434` replacing your original 
Alchemy, Infura, QuickNode, Llamanode or your own Ethereum-like node URL.

#### Debug

To debug your calls add `?debug=true` in the **sjrpc** URL: `http://localhost:8434?debug=true`

#### Different chainId

To call different chainId of what is defined in *SJRPC_URL* you need to add rpcUrl and chainId parameters in sjrpc URL.
Example: `?chainId=1337&rpcUrl=http://localhost:8545`

### Clean Up

When you need to call another Blockchain network your cache gets outdate and you need to clean it up. To do so you need to call the `cleanup` endpoint:

http://localhost:8434/cleanup

of via Make

```bash
make clean 
```

## Perfomance hint

It runs better in 64-bit architect processors, such M1/M2 Apple chips, or Intel i7. The reason is it uses Blake2b 512 bits.

### Acknowledgments

Thanks to @semaraugusto for helping to fix ethers calls.

Built with ❤️ using [Go / Golang](https://golang.dev)