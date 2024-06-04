# Miscellaneous Notes

OPC UA client library: https://github.com/gopcua/opcua


Not sure it works:
OPC Server: opc.tcp://opcuaserver.com:48484

OPC UA Information

NodeIDs:
https://opclabs.doc-that.com/files/onlinedocs/QuickOpc/Latest/User%27s%20Guide%20and%20Reference-QuickOPC/OPC%20UA%20Node%20IDs.html

OPC Client Library:
https://github.com/gopcua/opcua/blob/main/README.md


## Local OPC UA Server Prerequisits

https://github.com/FreeOpcUa/opcua-asyncio?tab=readme-ov-file

```
python -m venv venv 

source venv/bin/activate

pip install asyncua

```

## Start Local OPC UA Server

`python opcserver.py`

## Run Viam Module Standalone

`./bin/remoteserver opc-ua-sensor opc.tcp://0.0.0.0:4840/freeopcua/server/ "[\"ns=2;i=3\"]"`

