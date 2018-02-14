# BIP16 value checker

This tool can be used to find the BIP16 activation/enforcement block for both Bitcoin and Litecoin. You will need to run the corresponding daemon to retrieve the block information. An example can be found below for Bitcoin and Litecoin.

## Bitcoin

Values can be cross-checked here:

https://github.com/bitcoin/bitcoin/blob/master/src/chainparams.cpp#L78
Bitcoin BIP16 height: 173805
https://github.com/bitcoin/bitcoin/blob/0.15/src/validation.cpp#L1608
Bitcoin BIP16 switch time: 1333238400

```bash
bip16.exe --bip16target=1333238400 -block=173800 -rpcport=8332
2018/02/14 14:26:51 RPC URL: http://user:pass@127.0.0.1:8332
2018/02/14 14:26:51 Current block height: 502042
2018/02/14 14:26:51 Checking for BIP16 target block timestamp >= 1333238400
2018/02/14 14:26:51 Block: 00000000000000ce80a7e057163a4db1d5ad7b20fb6f598c9597b9665c8fb0d4 height: 173805 time: 1333240980 which has >= BIP16 target timestamp 1333238400

```

## Litecoin

Values can be cross-checked here:

https://github.com/litecoin-project/litecoin/blob/0.15/src/validation.cpp#L1608
Litecoin BIP16 switch time: 1349049600

```bash
>bip16.exe
2018/02/14 14:17:13 RPC URL: http://user:pass@127.0.0.1:9332
2018/02/14 14:17:13 Current block height: 1368352
2018/02/14 14:17:13 Checking for BIP16 target block timestamp >= 1349049600
2018/02/14 14:17:13 Block: 87afb798a3ad9378fcd56123c81fb31cfd9a8df4719b9774d71730c16315a092 height: 218579 time: 1349049710 which has >= BIP16 target timestamp 1349049600

```

## Usage

This tool supports the following parameters:

```bash
Usage of bip16.exe:
  -bip16target int
        Target timestamp for BIP16 activation. (default 1349049600)
  -block int
        Block height to start checking from. (default 218570)
  -rpchost string
        The RPC host to connect to. (default "127.0.0.1")
  -rpcpass string
        The RPC password. (default "pass")
  -rpcport int
        The RPC port to connect to. (default 9332)
  -rpcuser string
        The RPC username. (default "user")
  -verbose
        Toggle verbose reporting.
```