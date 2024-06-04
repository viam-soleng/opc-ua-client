# OPC Server Simulator

A very simplistic OPC Server setup for test and development purposes.

## Setup and Installation

```
python3.11 -m venv venv 

source venv/bin/activate

pip install asyncua

```

## Run the OPC Server

```
python opc-server/opcserver.py
```

## Run the module locally

```
./bin/remoteserver opc-ua-sensor opc.tcp://0.0.0.0:4840/freeopcua/server/ "[\"ns=2;i=3\"]"
```
