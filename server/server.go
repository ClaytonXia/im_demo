package server

import (
	"net/http"
	"sync"

	"github.com/claytonxia/im_demo/util"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// IMServer _
type IMServer struct {
	connMap map[string]*IMConnection
	mutex   sync.RWMutex
}

var (
	wsUpgrader websocket.Upgrader
)

const (
	// ConnIDLen _
	ConnIDLen = 10
	// ServerAddr _
	ServerAddr = "0.0.0.0:7777"
	// ReadQueueLen _
	ReadQueueLen = 1000
	// WriteQueueLen _
	WriteQueueLen = 1000
)

func init() {
	wsUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

// NewIMServer _
func NewIMServer() (server *IMServer) {
	server = &IMServer{
		connMap: make(map[string]*IMConnection),
	}
	return
}

// Start _
func (s *IMServer) Start() {
	http.HandleFunc("/ws", s.wsHandler)
	http.ListenAndServe(ServerAddr, nil)
}

func (s *IMServer) wsHandler(resp http.ResponseWriter, req *http.Request) {
	wsSocket, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		logrus.Error(err.Error())
		return
	}
	connID, err := util.RandID(ConnIDLen)
	if err != nil {
		logrus.Error(err.Error())
		return
	}
	logrus.Println("conn id:", connID)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, exists := s.connMap[connID]; exists {
		logrus.Errorf("repeat conn id: %s", connID)
		return
	}

	wsConn := &IMConnection{
		ID:        connID,
		WsSocket:  wsSocket,
		Server:    s,
		InChan:    make(chan *wsMessage, ReadQueueLen),
		OutChan:   make(chan *wsMessage, WriteQueueLen),
		CloseChan: make(chan byte),
	}
	s.connMap[connID] = wsConn

	go wsConn.refreshMsg()
	go wsConn.heartbeat()
	go wsConn.readTask()
	go wsConn.writeTask()
}

// Send _
func (s *IMServer) Send(idStr string, message []byte) {
	logrus.Printf("sending to: %s, %s\n", idStr, string(message))
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if tmpConn, exists := s.connMap[idStr]; exists {
		err := tmpConn.write(websocket.TextMessage, message)
		if err != nil {
			logrus.Errorf("conn write err: %s", err.Error())
		}
	} else {
		logrus.Fatal("nil client conn")
	}

	logrus.Printf("sending to: %s complete", idStr)
}

// Broadcast _
func (s *IMServer) Broadcast(selfConnID string, message []byte) {
	logrus.Printf("broadcasting message: %s", string(message))
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, tmpConn := range s.connMap {
		if tmpConn.ID == selfConnID {
			continue
		}
		err := tmpConn.write(websocket.TextMessage, message)
		if err != nil {
			logrus.Errorf("conn write err: %s", err.Error())
		}
	}
}

// ConnList _
// 在线列表
func (s *IMServer) ConnList(selfID string) (connList []string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, tmpConn := range s.connMap {
		if tmpConn.ID != selfID {
			connList = append(connList, tmpConn.ID)
		}
	}

	return
}

// RemoveConn _
func (s *IMServer) RemoveConn(connID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.connMap, connID)
}

// Stop _
func (s *IMServer) Stop() {
	logrus.Println("server stopping...")
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, conn := range s.connMap {
		conn.close()
	}

	logrus.Println("server stopped")
}
