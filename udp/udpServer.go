package udp

import (
	"github.com/op/go-logging"
	"net"
	"sync"
	"github.com/Catofes/go-routecontrol/config"
	"strconv"
)

var log *logging.Logger

type PackageType byte

var Server *MainUdpServer

type Handler func(*net.UDPConn, *net.UDPAddr, int, []byte)

type MainUdpServer struct {
	ListenPort    int
	ListenAddress string
	buffer        []byte
	mutex         sync.Mutex
	handler       map[PackageType]Handler
	connection    *net.UDPConn
}

func (s *MainUdpServer) loadConfig() {
	c := config.GetInstance("")
	s.ListenAddress = c.UdpListenAddress
	s.ListenPort = int(c.UdpListenPort)
}

func (s *MainUdpServer) Init() *MainUdpServer {
	s.loadConfig()
	s.buffer = make([]byte, 1024)
	s.handler = make(map[PackageType]Handler, 5)
	return s
}

func (s *MainUdpServer) Loop() {
	address, err := net.ResolveUDPAddr("udp", s.ListenAddress+":"+strconv.Itoa(s.ListenPort))
	if err != nil {
		log.Fatal("Can't resolve address: ", err)
	}
	connection, err := net.ListenUDP("udp", address)
	s.connection = connection
	if err != nil {
		log.Fatal("Can't listen udp on", address, err)
	}
	defer s.connection.Close()
	for {
		s.handleClient(s.connection)
	}
}

func (s *MainUdpServer) handleClient(connection *net.UDPConn) {
	n, remoteAddress, err := connection.ReadFromUDP(s.buffer)
	if err != nil {
		log.Warning("Error read connection. %s", err.Error())
		return
	}
	//log.Debug("Get connection from %s, size %d.", remoteAddress.String(), n)
	if n <= 0 {
		return
	}
	packageType := PackageType(s.buffer[0])
	s.mutex.Lock()
	if handler, ok := s.handler[packageType]; ok {
		s.mutex.Unlock()
		handler(connection, remoteAddress, n, s.buffer)
	} else {
		s.mutex.Unlock()
		log.Warning("Receive unknow package.")
	}
}

func (s *MainUdpServer) AddHandler(package_type PackageType, handler Handler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handler[package_type] = handler
}

func (s *MainUdpServer) DeleteHandler(package_type PackageType) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.handler, package_type)
}

func Run() {
	Server = (&MainUdpServer{}).Init()
	go Server.Loop()
}
