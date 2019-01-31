package udp

import (
	"github.com/emirpasic/gods/maps/treemap"
	"encoding/binary"
	"net"
	"time"
	"sync"
)

const PingRequestPackageType = PackageType(0)
const PingReplyPackageType = PackageType(1)
const PingPackageLength = 29

type PingPackage struct {
	Type             PackageType
	NodeId           int32
	Id               int64
	RequestTimestamp int64
	ReplyTimestamp   int64
	Relay            int64
}

func (s *PingPackage) ToBytes() ([]byte) {
	data := make([]byte, PingPackageLength)
	data[0] = byte(s.Type)
	binary.BigEndian.PutUint32(data[1:5], uint32(s.NodeId))
	binary.BigEndian.PutUint32(data[5:13], uint32(s.Id))
	binary.BigEndian.PutUint64(data[13:21], uint64(s.RequestTimestamp))
	binary.BigEndian.PutUint64(data[21:29], uint64(s.ReplyTimestamp))
	return data
}

func (s *PingPackage) FromBytes(data []byte) (*PingPackage) {
	s.Type = PackageType(data[0])
	s.NodeId = int32(binary.BigEndian.Uint32(data[1:5]))
	s.Id = int64(binary.BigEndian.Uint64(data[5:13]))
	s.RequestTimestamp = int64(binary.BigEndian.Uint64(data[13:21]))
	s.ReplyTimestamp = int64(binary.BigEndian.Uint64(data[21:29]))
	return s
}

func PingRequestHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	if n != PingPackageLength {
		log.Info("Wrong package size at ping request package.")
		return
	}
	pingPackage := (&PingPackage{}).FromBytes(data)
	pingPackage.ReplyTimestamp = time.Now().UnixNano()
	pingPackage.Type = PingReplyPackageType
	data = pingPackage.ToBytes()
	n, err := conn.WriteToUDP(data, addr)
	if err != nil || n != PingPackageLength {
		log.Info("Write ping reply package to %s wrong", addr.String())
	}
}

type PingStack struct {
	data                 map[int]*treemap.Map
	latency              map[int]int64
	receivedPackageCount map[int]int64
	packageLost          map[int]float64
	mutex                sync.Mutex
}

func (s *PingStack) Init() *PingStack {
	s.data = make(map[int]*treemap.Map)
	s.latency = make(map[int]int64)
	s.receivedPackageCount = make(map[int]int64)
	s.packageLost = make(map[int]float64)
	return s
}

func (s *PingStack) CreateNode(nodeId int) {
	if _, ok := s.data[nodeId]; ok {
	} else {
		s.data[nodeId] = treemap.NewWithIntComparator()
	}
}

func (s *PingStack) CheckFullStack(nodeId int) {
	if val, ok := s.data[nodeId]; ok {
		if val.Size() > 1000 {
			k, v := val.Min()
			p := v.(*PingPackage)
			if p.ReplyTimestamp > 0 {
				s.receivedPackageCount[nodeId]--
			}
			val.Remove(k)
		}
	}
}

func (s *PingStack) Get(nodeId int) *PingPackage {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if val, ok := s.data[nodeId]; ok {
		var id int64 = 0
		if tmp, _ := val.Max(); tmp != nil {
			id = tmp.(int64) + 1
		}
		pingPackage := PingPackage{}
		pingPackage.Id = id
		pingPackage.RequestTimestamp = time.Now().UnixNano()
		val.Put(id, &pingPackage)
		s.CheckFullStack(nodeId)
		s.packageLost[nodeId] = 1 - float64(s.receivedPackageCount[nodeId])/float64(val.Size()-1)
		return &pingPackage
	}
	return nil
}

func (s *PingStack) Put(reply *PingPackage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	nodeId := int(reply.NodeId)
	if server, ok := s.data[nodeId]; ok {
		id := reply.Id
		if v, ok := server.Get(id); ok {
			request := v.(*PingPackage)
			if request.ReplyTimestamp > 0 {
				return
			}
			request.ReplyTimestamp = reply.ReplyTimestamp
			request.Relay = time.Now().UnixNano() - request.RequestTimestamp
			s.latency[nodeId] = (s.latency[nodeId]*s.receivedPackageCount[nodeId] + request.Relay) /
				(s.receivedPackageCount[nodeId] + 1)
			s.receivedPackageCount[nodeId]++
			s.CheckFullStack(nodeId)
			s.packageLost[nodeId] = 1 - float64(s.receivedPackageCount[nodeId])/float64(server.Size())
		}
	}
}

func (s *PingStack) GetServerInfo(nodeId int) (latency int64, packageLost float64) {
	if _, ok := s.data[nodeId]; ok {
		return s.latency[nodeId], s.packageLost[nodeId]
	}
	return -1, 0
}

func (s *PingStack) PingReplyHandler(conn *net.UDPConn, addr *net.UDPAddr, n int, data []byte) {
	if n != PingPackageLength {
		log.Info("Wrong package size at ping request package.")
		return
	}
	pingPackage := (&PingPackage{}).FromBytes(data)
	s.Put(pingPackage)
}
