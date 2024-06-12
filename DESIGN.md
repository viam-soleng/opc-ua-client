# Design Notes for this Release

## Goals

- Reduce data volume transmitted and collected by only recording the changes by using subscribe pattern
- Changes/events will be stored in a buffer and continuously read/polled by data manager starting with the oldest item
- Read calls from other clients will always return the most recent information

## Configuration

```json
{
    "endpoint": "opc.tcp://0.0.0.0:4840/freeopcua/server/",
    "subscribe": "data" | "event" | "poll", // default is poll
    "nodeids": [
        "ns=2;i=3",
        "ns=2;i=2"
    ]
}
```


## Data Flow

1. Subscribe to nodes
2. Start go routine(s) to process change notifications/events
3. Make read api read from buffer
4. If read call comes from data manager, read oldest item not yet read by data manager
5. Mark the read item in the buffer as read by data manager
