# State Sync

## Manager

Manager implements Reactor of Network module

```mermaid
classDiagram
    direction LR
    class Reactor {
        <<interface>>
        +OnReceive(pi ProtocolInfo, b []byte, id PeerID) (bool, error)
        +OnFailure(err error, pi ProtocolInfo, b []byte)
        +OnJoin(id PeerID)
        +OnLeave(id PeerID)
    }
    class Syncer {
        <<interface>>
        +ForceSync() (*Result, error)
        +Stop()
        +Finalize() error
    }
    class Platform {
        <<interface>>
        +NewExtensionWithBuilder(builder, raw) ExtensionSnapshot
    }
    %% pi: protocol info
    %% b: byte
    %% ah: account hash
    %% prh: patch tx receipts hash
    %% nrh: normal tx receipts hash
    %% vh: validator list hash
    %% ed: extenstion data
    class Manager {
        <<struct>>
        log Logger
        pool *peerPool
        server *server
        client *client
        db Database
        syncing bool
        syncer *syncer
        mutex Mutex
        plt Platform

        +OnReceive(pi, b, id) (bool, error)
        +OnFailure(err, pi, b)
        +OnJoin(id)
        +OnLeave(id)
        +NewSyncer(ah []byte, prh []byte, nrh []byte, vh []byte, ed []byte)
    }
    class Result {
        <<struct>>
        Wss            WorldSnapshot
        PatchReceipts  ReceiptList
        NormalReceipts ReceiptList
    }
    Reactor <|-- Manager : implements
```

## Syncer

syncer implements Syncer of Manager interface and Callback interface

```mermaid
classDiagram
    class Syncer {
        <<interface>>
        +ForceSync() (*Result, error)
        +Stop()
        +Finalize() error
    }
    class Callback {
        <<interface>>
        onResult(status errCode, p *peer)
        onNodeData(p *peer, status errCode, t syncType, data [][]byte)
        onReceive(pi ProtocolInfo, b []byte, p *peer)
    }
    class syncer {
        <<struct>>
        mutex Mutex
        cond *Cond
        client *client
        database Database
        plt Platform

        pool *peerPool
        vpool *peerPool
        ivpool *peerPool
        sentReqer map[peerID]*peer
        reqValue [4]map[string]bool
        
        builder [4]Builder
        bMutex [4]Mutex
        rPeerCnt [4]int

        ah []byte
        vlh []byte
        ed []byte
        prh []byte
        nrh []byte

        finishCh chan error
        log Logger
        
        wss WorldSnapshot
        prl ReceiptList
        nrl ReceiptList
        cb func

        waitingPeerCnt int
        complete syncType
        startTime Time

        Complete(st SyncType)
        Stop()
        ForceSync() : (*Result, error)
        Finalize() error

        _reqUnresolvedNode(st syncType, builder Builder, need int)
        _onNodeData(p *peer, status errCode, st syncType, data [][]byte)
        _updateValidPool()
        _requestIfNotEnough(p *peer)
        _returnPeers(peers *peer)
        _reservePeers(need int, st syncType) : ([]*peer)

        reqUnresolved(sy syncType, builder Builder, need int)
        onNodeData(p *peer, status errCode, st syncType, data [][]byte)
        onReceive(pi ProtocolInfo, b []byte, p *peer)
        onJoin(p *peer)
        onLeave(id PeerID)
        processMsg(pi ProtocolInfo, b []byte, p *peer)
        onResult(status errCode, p *peer)
    }

    Callback <|-- syncer : implements
    Syncer <|-- syncer : implements

```

## client
```mermaid
classDiagram
    class client {
        <<struct>>
        ph ProtocolHandler
        mutex Mutex
        log Logger

        hasNode(p *peer, wsHash []byte, prHash []byte, nrHash []byte, vh []byte, expiredCb func(...))
        requestNodeData(p *peer, hash [][]byte, t syncType, expiredCb func(...))
    }
```

## server
```mermaid
classDiagram
    class server {
        <<struct>>
        database Database
        ph ProtocolHandler
        log Logger
        merkleTrie Bucket
        bytesByHash Bucket

        _resolveNode(hashed [][]byte)
        onReceive(pi ProtocolInfo, b []byte, p *peer)
        hasNode(msg []byte, p *peer)
        requestNode(msg []byte, p *peer)
    }
```

## protocol

 4 protocols defined
  - protoHasNode
  - protoResult
  - protoRequestNodeData
  - protoNodeData

```mermaid
classDiagram
    class hasNode {
        <<struct>>
        ReqID uint32
        StateHash []byte
        ValidatorHash []byte
        PatchHash []byte
        NormalHash []byte
    }

    class result {
        <<struct>>
        ReqId uint32
        Status errCode
    }

    class requestNodeData {
        <<struct>>
        ReqID uint32
        Type syncType
        Hashed [][]byte
    }

    class nodeData {
        <<struct>>
        ReqID uint32
        Status errCode
        Type syncType
        Data [][]byte
    }
```

## peer
```mermaid
classDiagram
    class peer {
        <<struct>>
        id PeerID
        reqID uint32
        expired Duration
        timer Timer
        cb Callback
        log Logger

        onReceive(pi ProtocolInfo, data interface)
        String()
    }
    class peerPool {
        <<struct>>
        ch chan PeerID
        peers map[PeerID]*Element
        pList *List
        log Logger

        push(p *peer)
        size()
        pop()
        remove(id PeerID)
        getPeer(id PeerID)
        peerList()
    }
```

## Sequence diagram

### Force State Sync
```mermaid
sequenceDiagram
    autonumber
    participant Transition
    participant Syncer
    participant Client
    participant Server
    Transition->>Syncer: ForceSync
    loop peerlist
        Syncer->>Client: hasNode
        Note over Client: protoHasNode
        Client->>Server: hasNode
    end
    Note over Server: protoResult
    Server-->>Syncer: processMsg
    par syncExtensionState
        loop peerlist
            Syncer->>Client: requestNodeData
            Note over Client: protoRequestNodeData
            Client->>Server: requestNode
        end
        Note over Server: protoNodeData
        Server-->>Syncer: processMsg
    and SyncWorldState
        loop peerlist
            Syncer->>Client: requestNodeData
            Note over Client: protoRequestNodeData
            Client->>Server: requestNode
        end
        Note over Server: protoNodeData
        Server-->>Syncer: processMsg
    and SyncPatchReceipts
        loop peerlist
            Syncer->>Client: requestNodeData
            Note over Client: protoRequestNodeData
            Client->>Server: requestNode
        end
        Note over Server: protoNodeData
        Server-->>Syncer: processMsg
    and SyncNormalReceipts
        loop peerlist
            Syncer->>Client: requestNodeData
            Note over Client: protoRequestNodeData
            Client->>Server: requestNode
        end
        Note over Server: protoNodeData
        Server-->>Syncer: processMsg
    end
    Syncer->>Transition: Result
```
