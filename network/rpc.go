package network

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/icon-project/goloop/module"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"
)

func MethodRepository(nm module.NetworkManager) *jsonrpc.MethodRepository {
	mr := jsonrpc.NewMethodRepository()
	m := nm.(*manager)
	//RegisterMethod(method string, h Handler, params, result interface{}) error
	_ = mr.RegisterMethod("dial", jsonrpcWithContext(m, jsonrpcHandleDial), nil, nil)
	_ = mr.RegisterMethod("query", jsonrpcWithContext(m, jsonrpcHandleSendQuery), nil, nil)
	_ = mr.RegisterMethod("p2p", jsonrpcWithContext(m, jsonrpcHandleP2P), nil, nil)
	_ = mr.RegisterMethod("protocol", jsonrpcWithContext(m, jsonrpcHandleProtocol), nil, nil)
	_ = mr.RegisterMethod("logger", jsonrpcWithContext(m, jsonrpcHandleLogger), nil, nil)
	return mr
}

type jsonrpcHandlerFunc func(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error)

func (f jsonrpcHandlerFunc) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	return f(c, params)
}

type rpcContextKey string

var (
	rpcContextKeyParamMap = rpcContextKey("param")
	rpcContextKeyManager  = rpcContextKey("manager")
)

func jsonrpcWithContext(mgr *manager, next jsonrpcHandlerFunc) jsonrpcHandlerFunc {
	return func(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
		m := make(map[string]interface{})
		if err := jsonrpc.Unmarshal(params, &m); err != nil {
			log.Println("jsonrpcWithChannel jsonrpc.Unmarshal", err)
		}
		ctx := context.WithValue(c, rpcContextKeyParamMap, m)
		//log.Println("jsonrpcWithChannel param", m)

		ctx = context.WithValue(ctx, rpcContextKeyManager, mgr)
		return next.ServeJSONRPC(ctx, params)
	}
}

func _getParamString(c context.Context, k string) (string, *jsonrpc.Error) {
	m, err := _getParamMap(c)
	if err != nil {
		return "", err
	}
	v, ok := m[k]
	if !ok {
		log.Println("_getParamString not exists", k)
		return "", jsonrpc.ErrInvalidParams()
	}
	s, ok := v.(string)
	if !ok || s == "" {
		log.Println("_getParamString invalid param value to string")
		return "", jsonrpc.ErrInvalidParams()
	}
	return s, nil
}

func _getParamStringArray(c context.Context, k string) ([]string, *jsonrpc.Error) {
	m, err := _getParamMap(c)
	if err != nil {
		return nil, err
	}
	v, ok := m[k]
	if !ok {
		log.Println("_getParamStringArray not exists", k)
		return nil, jsonrpc.ErrInvalidParams()
	}
	a, ok := v.([]interface{})
	if !ok {
		log.Printf("_getParamStringArray invalid param value to []interface{} from %#v", v)
		return nil, jsonrpc.ErrInvalidParams()
	}
	arr := make([]string, len(a))
	for i, e := range a {
		s, ok := e.(string)
		if !ok {
			log.Printf("_getParamStringArray invalid param value to string from %#v", e)
			return nil, jsonrpc.ErrInvalidParams()
		}
		arr[i] = s
	}
	return arr, nil
}

func _getParamMap(c context.Context) (map[string]interface{}, *jsonrpc.Error) {
	v := c.Value(rpcContextKeyParamMap)
	if v == nil {
		log.Println("_getParamMap not exists rpcContextKeyParamMap")
		return nil, jsonrpc.ErrInvalidParams()
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		log.Println("_getParamMap invalid context value to map[string]interface{}")
		return nil, jsonrpc.ErrInvalidParams()
	}
	return m, nil
}

func _getManager(c context.Context) (*manager, *jsonrpc.Error) {
	v := c.Value(rpcContextKeyManager)
	if v == nil {
		log.Println("_getManager not exists rpcContextKeyManager")
		return nil, jsonrpc.ErrInvalidParams()
	}
	mgr, ok := v.(*manager)
	if !ok {
		log.Println("_getManager invalid context value to *manager")
		return nil, jsonrpc.ErrInternal()
	}
	return mgr, nil
}

func _getP2P(c context.Context) (*PeerToPeer, *jsonrpc.Error) {
	mgr, err := _getManager(c)
	if err != nil {
		return nil, err
	}
	return mgr.p2p, nil
}
func jsonrpcHandleSendQuery(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}
	id, err := _getParamString(c, "id")
	if err != nil {
		return nil, err
	}
	p := p2p.getPeer(NewPeerIDFromString(id), false)
	p2p.sendQuery(p)
	return nil, nil
}
func jsonrpcHandleDial(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}
	addr, err := _getParamString(c, "addr")
	if err != nil {
		return nil, err
	}
	dErr := p2p.dial(NetAddress(addr))
	if dErr != nil {
		log.Println("jsonrpcHandleDial dial fail", dErr.Error())
		return nil, jsonrpc.ErrInternal()
	}
	return nil, nil
}
func jsonrpcHandleP2P(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	m["self"] = peerToMap(p2p.self)
	m["seeds"] = p2p.seeds.Map()
	m["roots"] = p2p.roots.Map()
	m["friends"] = peerToMapArray(p2p.friends)
	m["parent"] = peerToMap(p2p.parent)
	m["children"] = peerToMapArray(p2p.children)
	m["uncles"] = peerToMapArray(p2p.uncles)
	m["nephews"] = peerToMapArray(p2p.nephews)
	m["orphanages"] = peerToMapArray(p2p.orphanages)
	return m, nil
}

func jsonrpcHandleProtocol(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	mgr, err := _getManager(c)
	if err != nil {
		return nil, err
	}
	protocol, _ := _getParamString(c, "protocol")

	m := make(map[string]interface{})
	if ph, ok := mgr.protocolHandlers[protocol];ok {
		m[ph.name] = protocolHandlerToMap(ph)
	}else{
		for _, ms := range mgr.protocolHandlers {
			m[ms.name] = protocolHandlerToMap(ms)
		}
	}
	return m, nil
}

func jsonrpcHandleLogger(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {
	p2p, err := _getP2P(c)
	if err != nil {
		return nil, err
	}
	logger, err := _getParamString(c, "logger")
	if err != nil {
		return nil, err
	}
	excludes, err := _getParamStringArray(c, "excludes")
	if err != nil {
		return nil, err
	}
	switch logger {
	case "global":
		singletonLoggerExcludes = excludes[:]
	case "PeerToPeer":
		p2p.log.excludes = excludes[:]
		//NetworkManager
		//ProtocolHandler
		//Transport
		//Listener
		//Dialer
		//PeerDispatcher
		//ChannelNegotiator
		//Authenticator
	default:
		// ignore
	}
	return excludes, nil
}

func peerToMapArray(s *PeerSet) []map[string]interface{} {
	rarr := make([]map[string]interface{}, s.Len())
	for i, v := range s.Array() {
		rarr[i] = peerToMap(v)
	}
	return rarr
}
func peerToMap(p *Peer) map[string]interface{} {
	m := make(map[string]interface{})
	if p != nil {
		m["id"] = p.id.String()
		m["addr"] = p.netAddress
		m["in"] = p.incomming
		m["channel"] = p.channel
		m["role"] = p.role
		m["conn"] = p.connType
		m["rtt"] = p.rtt.String()
		if p.q != nil {
			sq := make([]string,DefaultSendQueueMaxPriority)
			for i:=0;i<DefaultSendQueueMaxPriority;i++{
				sq[i] = strconv.Itoa(p.q.Available(i))
			}
			m["sendQueue"] = strings.Join(sq,",")
		}
	}
	return m
}
func protocolHandlerToMap(ph *protocolHandler) map[string]interface{} {
	m := make(map[string]interface{})
	if ph != nil {
		m["protocol"] = fmt.Sprintf("%#04x,", ph.protocol.Uint16())
		m["priority"] = ph.priority

		parr := make([]int, 0)
		for _, p := range ph.subProtocols {
			parr = append(parr, int(p.Uint16()))
		}
		sort.Ints(parr)
		sarr := make([]string, len(parr))
		for i, p := range parr {
			sarr[i] = fmt.Sprintf("%#04x", p)
		}
		m["subProtocols"] = strings.Join(sarr, ",")

		m["receiveQueue"] = ph.receiveQueue.Available()
		m["eventQueue"] = ph.eventQueue.Available()
		m["sendQueue"] = ph.m.p2p.sendQueue.Available(int(ph.protocol.ID()))
	}
	return m
}
