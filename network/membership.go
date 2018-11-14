package network

import (
	"container/list"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type membership struct {
	name        string
	protocol    module.ProtocolInfo
	peerToPeer  *PeerToPeer
	roles       map[module.Role]*PeerIdList
	authorities map[module.Authority]*RoleList
	reactors    map[string]module.Reactor
	cbFuncs     map[module.ProtocolInfo]receiveCbFunc
}

type receiveCbFunc func(pi module.ProtocolInfo, bytes []byte, peerId module.PeerId) (bool, error)

func newMembership(name string, pi module.ProtocolInfo, p2p *PeerToPeer) module.Membership {
	m := &membership{
		name:        name,
		protocol:    pi,
		peerToPeer:  p2p,
		roles:       make(map[module.Role]*PeerIdList),
		authorities: make(map[module.Authority]*RoleList),
		reactors:    make(map[string]module.Reactor),
		cbFuncs:     make(map[module.ProtocolInfo]receiveCbFunc),
	}
	p2p.setPacketCbFunc(pi, m.onPacket)
	return m
}

//TODO using worker pattern {pool or each packet or none} for reactor
func (m *membership) workerRoutine() {

}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (m *membership) onPacket(pkt *Packet, p *Peer) {
	//Check authority
	//roles := Roles(pkt.src)
	//auth := Authority(pkt.cast)
	//r := HasAuthority(auth, role) range roles
	//if r == true

	if cbFunc := m.cbFuncs[pkt.subProtocol]; cbFunc != nil {
		r, err := cbFunc(pkt.subProtocol, pkt.payload, p.Id())
		if err != nil {
			log.Println(err)
		}
		if r {
			m.peerToPeer.ch <- pkt
		}
	}
}

func (m *membership) RegistReactor(name string, reactor module.Reactor, subProtocols []module.ProtocolInfo) error {
	if _, ok := m.reactors[name]; ok {
		return common.ErrIllegalArgument
	}
	for _, sp := range subProtocols {
		if _, ok := m.cbFuncs[sp]; ok {
			return common.ErrIllegalArgument
		}
		m.cbFuncs[sp] = reactor.OnReceive
	}
	return nil
}

func (m *membership) Unicast(subProtocol module.ProtocolInfo, bytes []byte, peerId module.PeerId) error {
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = PROTO_DEF_MEMBER
	return nil
}

//TxMessage,VoteMessage, Send to Validators
func (m *membership) Multicast(subProtocol module.ProtocolInfo, bytes []byte, role module.Role) error {
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = PROTO_DEF_MEMBER
	m.peerToPeer.sendToUpside(pkt)
	return nil
}

//ProposeMessage,BlockMessage, Send to Citizen
func (m *membership) Broadcast(subProtocol module.ProtocolInfo, bytes []byte, broadcastType module.BroadcastType) error {
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = PROTO_DEF_MEMBER
	pkt.ttl = byte(broadcastType)
	m.peerToPeer.sendToFriends(pkt)
	m.peerToPeer.sendToDownside(pkt)
	return nil
}

func (m *membership) getRolePeerIdList(role module.Role) *PeerIdList {
	l, ok := m.roles[role]
	if !ok {
		l := NewPeerIdList()
		m.roles[role] = l
	}
	return l
}

func (m *membership) AddRole(role module.Role, peerId module.PeerId) error {
	l := m.getRolePeerIdList(role)
	l.PushBack(peerId)
	return nil
}

func (m *membership) RemoveRole(role module.Role, peerId module.PeerId) error {
	l := m.getRolePeerIdList(role)
	l.Remove(peerId)
	return nil
}

func (m *membership) HasRole(role module.Role, peerId module.PeerId) bool {
	l := m.getRolePeerIdList(role)
	return l.Has(peerId)
}

func (m *membership) Roles(peerId module.PeerId) []module.Role {
	var i int
	s := make([]module.Role, 0, len(m.roles))
	for k, v := range m.roles {
		if v.Has(peerId) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}

func (m *membership) getAuthorityRoleList(authority module.Authority) *RoleList {
	l, ok := m.authorities[authority]
	if !ok {
		l := NewRoleList()
		m.authorities[authority] = l
	}
	return l
}

func (m *membership) GrantAuthority(authority module.Authority, role module.Role) error {
	l := m.getAuthorityRoleList(authority)
	l.PushBack(role)
	return nil
}

func (m *membership) DenyAuthority(authority module.Authority, role module.Role) error {
	l := m.getAuthorityRoleList(authority)
	l.Remove(role)
	return nil
}

func (m *membership) HasAuthority(authority module.Authority, role module.Role) bool {
	l := m.getAuthorityRoleList(authority)
	return l.Has(role)
}

func (m *membership) Authorities(role module.Role) []module.Authority {
	var i int
	s := make([]module.Authority, len(m.authorities))
	for k, v := range m.authorities {
		if v.Has(role) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}

///////
type StringList struct {
	*list.List
}

func NewStringList() *StringList {
	return &StringList{list.New()}
}

func (l *StringList) get(v string) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(string); s == v {
			return e
		}
	}
	return nil
}

func (l *StringList) Remove(v string) bool {
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *StringList) Has(v string) bool {
	return l.get(v) != nil
}

type PeerIdList struct {
	*list.List
}

func NewPeerIdList() *PeerIdList {
	return &PeerIdList{list.New()}
}

func (l *PeerIdList) get(v module.PeerId) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(module.PeerId); s == v {
			return e
		}
	}
	return nil
}

func (l *PeerIdList) Remove(v module.PeerId) bool {
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *PeerIdList) Has(v module.PeerId) bool {
	return l.get(v) != nil
}

func (l *PeerIdList) IsEmpty() bool {
	return l.Len() == 0
}

type RoleList struct {
	*list.List
}

func NewRoleList() *RoleList {
	return &RoleList{list.New()}
}

func (l *RoleList) get(v module.Role) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(module.Role); s == v {
			return e
		}
	}
	return nil
}

func (l *RoleList) Remove(v module.Role) bool {
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *RoleList) Has(v module.Role) bool {
	return l.get(v) != nil
}
