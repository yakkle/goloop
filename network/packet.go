package network

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	packetHeaderSize = 10 + peerIDSize
	packetFooterSize = 8
)

//srcPeerId, castType, destInfo, TTL(0:unlimited)
type Packet struct {
	protocol        protocolInfo //2byte
	subProtocol     protocolInfo //2byte
	src             module.PeerID       //20byte
	dest            byte
	ttl             byte
	lengthOfpayload uint32 //4byte
	payload         []byte
	hashOfPacket    uint64 //8byte
	//Transient fields
	sender module.PeerID //20byte
	destPeer module.PeerID //20byte
	priority uint8
}

const (
	p2pDestAny       = 0x00
	p2pDestPeerGroup = 0x08
	p2pDestPeer      = 0xFF
)

func NewPacket(pi protocolInfo, spi protocolInfo, payload []byte) *Packet {
	return &Packet{
		protocol:        pi,
		subProtocol:     spi,
		lengthOfpayload: uint32(len(payload)),
		payload:         payload[:],
	}
}

func newPacket(spi protocolInfo, payload []byte) *Packet {
	return NewPacket(PROTO_CONTOL, spi, payload)
}

func (p *Packet) String() string {
	return fmt.Sprintf("{pi:%#04x,subPi:%#04x,src:%v,dest:%#x,ttl:%d,len:%v,hash:%#x}",
		p.protocol.Uint16(),
		p.subProtocol.Uint16(),
		p.src,
		p.dest,
		p.ttl,
		p.lengthOfpayload,
		p.hashOfPacket)
}

type PacketReader struct {
	*bufio.Reader
	rd   io.Reader
	pkt  *Packet
	hash hash.Hash64
}

// NewReader returns a new Reader whose buffer has the default size.
func NewPacketReader(rd io.Reader) *PacketReader {
	return &PacketReader{Reader: bufio.NewReaderSize(rd, DefaultPacketBufferSize), rd: rd}
}

func (pr *PacketReader) _read(n int) ([]byte, error) {
	b := make([]byte, n)
	rn := 0
	for {
		tn, err := pr.Reader.Read(b[rn:])
		if err != nil {
			return nil, err
		}
		rn += tn
		if rn >= n {
			break
		}
	}
	return b, nil
}

func (pr *PacketReader) Reset() {
	pr.Reader.Reset(pr.rd)
}

func (pr *PacketReader) ReadPacket() (pkt *Packet, h hash.Hash64, e error) {
	for {
		if pr.pkt == nil {
			hb, err := pr._read(packetHeaderSize)
			if err != nil {
				e = err
				return
			}
			tb := hb[:]
			pi := newProtocolInfoFrom(tb[:2])
			tb = tb[2:]
			spi := newProtocolInfoFrom(tb[:2])
			tb = tb[2:]
			src := NewPeerID(tb[:peerIDSize])
			tb = tb[peerIDSize:]
			dest := tb[0]
			tb = tb[1:]
			ttl := tb[0]
			tb = tb[1:]
			lop := binary.BigEndian.Uint32(tb[:4])
			if lop > DefaultPacketPayloadMax {
				e = fmt.Errorf("invalid packet lengthOfpayload %x, max:%x", lop, DefaultPacketPayloadMax)
				return
			}
			tb = tb[4:]
			pr.pkt = &Packet{protocol: pi, subProtocol: spi, src: src, dest: dest, ttl: ttl, lengthOfpayload: lop}
			h = fnv.New64a()
			if _, err = h.Write(hb); err != nil {
				log.Printf("PacketReader.ReadPacket hash/fnv.Hash64.Write hb %T %#v %s", err, err, err)
			}
		}

		if pr.pkt.payload == nil {
			payload, err := pr._read(int(pr.pkt.lengthOfpayload))
			if err != nil {
				e = err
				return
			}
			pr.pkt.payload = payload
			if _, err = h.Write(payload); err != nil {
				log.Printf("PacketReader.ReadPacket hash/fnv.Hash64.Write payload %T %#v %s", err, err, err)
			}
		}

		if pr.pkt.hashOfPacket == 0 {
			fb, err := pr._read(packetFooterSize)
			if err != nil {
				e = err
				return
			}
			tb := fb[:]
			pr.pkt.hashOfPacket = binary.BigEndian.Uint64(tb[:8])
			tb = tb[8:]

			pkt = pr.pkt
			pr.pkt = nil
			return
		}
	}

}

type PacketWriter struct {
	*bufio.Writer
	wr io.Writer
}

func NewPacketWriter(w io.Writer) *PacketWriter {
	return &PacketWriter{Writer: bufio.NewWriterSize(w, DefaultPacketBufferSize), wr: w}
}

func (pw *PacketWriter) Reset() {
	pw.Writer.Reset(pw.wr)
}

func (pw *PacketWriter) WritePacket(pkt *Packet) error {
	hb := make([]byte, packetHeaderSize)
	tb := hb[:]
	pkt.protocol.Copy(tb[:2])
	tb = tb[2:]
	pkt.subProtocol.Copy(tb[:2])
	tb = tb[2:]
	pkt.src.Copy(tb[:peerIDSize])
	tb = tb[peerIDSize:]
	tb[0] = pkt.dest
	tb = tb[1:]
	tb[0] = pkt.ttl
	tb = tb[1:]
	binary.BigEndian.PutUint32(tb[:4], pkt.lengthOfpayload)
	tb = tb[4:]
	_, err := pw.Write(hb)
	if err != nil {
		log.Printf("PacketWriter.WritePacket hb %T %#v %s", err, err, err)
		return err
	}
	//
	payload := pkt.payload[:pkt.lengthOfpayload]
	_, err = pw.Write(payload)
	if err != nil {
		log.Printf("PacketWriter.WritePacket payload %T %#v %s", err, err, err)
		return err
	}
	//
	fb := make([]byte, packetFooterSize)
	tb = fb[:]
	if pkt.hashOfPacket == 0 {
		h := fnv.New64a()
		if _, err = h.Write(hb); err != nil {
			log.Printf("PacketWriter.WritePacket hash/fnv.Hash64.Write hb %T %#v %s", err, err, err)
			return err
		}
		if _, err = h.Write(payload); err != nil {
			log.Printf("PacketWriter.WritePacket hash/fnv.Hash64.Write payload %T %#v %s", err, err, err)
			return err
		}
		pkt.hashOfPacket = h.Sum64()
	}
	binary.BigEndian.PutUint64(tb[:8], pkt.hashOfPacket)
	tb = tb[8:]
	_, err = pw.Write(fb)
	if err != nil {
		log.Printf("PacketWriter.WritePacket fb %T %#v %s", err, err, err)
		return err
	}
	return nil
}

func (pw *PacketWriter) Write(b []byte) (int, error) {
	wn := 0
	re := 0
	for {
		n, err := pw.Writer.Write(b[wn:])
		wn += n
		if err != nil && err == io.ErrShortWrite && re < DefaultPacketRewriteLimit {
			re++
			log.Println("PacketWriter.Write io.ErrShortWrite", err)
			time.Sleep(DefaultPacketRewriteDelay)
			continue
		} else {
			return wn, err
		}
	}
}

func (pw *PacketWriter) Flush() error {
	re := 0
	for {
		err := pw.Writer.Flush()
		if err != nil && err == io.ErrShortWrite && re < DefaultPacketRewriteLimit {
			re++
			log.Println("PacketWriter.Flush io.ErrShortWrite", err)
			time.Sleep(DefaultPacketRewriteDelay)
			continue
		} else {
			return err
		}
	}
}

type PacketReadWriter struct {
	b    *bytes.Buffer
	rd   *PacketReader
	wr   *PacketWriter
	rpkt *Packet
	wpkt *Packet
	mtx  sync.RWMutex
}

func NewPacketReadWriter() *PacketReadWriter {
	b := bytes.NewBuffer(make([]byte, DefaultPacketBufferSize))
	b.Reset()
	return &PacketReadWriter{b: b, rd: NewPacketReader(b), wr: NewPacketWriter(b)}
}

func (prw *PacketReadWriter) WritePacket(pkt *Packet) error {
	defer prw.mtx.Unlock()
	prw.mtx.Lock()
	if err := prw.wr.WritePacket(pkt); err != nil {
		return err
	}
	if err := prw.wr.Flush(); err != nil {
		return err
	}
	prw.wpkt = pkt
	return nil
}

func (prw *PacketReadWriter) ReadPacket() (*Packet, error) {
	defer prw.mtx.RUnlock()
	prw.mtx.RLock()
	if prw.rpkt == nil {
		//(pkt *Packet, h hash.Hash64, e error)
		pkt, h, err := prw.rd.ReadPacket()
		if err != nil {
			return nil, err
		}
		if pkt.hashOfPacket != h.Sum64() {
			err := fmt.Errorf("invalid hashOfPacket:%x, expected:%x", pkt.hashOfPacket, h.Sum64())
			return pkt, err
		}
		prw.rpkt = pkt
	}
	return prw.rpkt, nil
}

func (prw *PacketReadWriter) Reset() {
	defer prw.mtx.Unlock()
	prw.mtx.Lock()
	prw.b.Reset()
	prw.rd.Reset()
	prw.wr.Reset()
	prw.rpkt = nil
	prw.wpkt = nil
}
