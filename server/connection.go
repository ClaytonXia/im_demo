package server

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

// IMConnection _
type IMConnection struct {
	ID       string
	WsSocket *websocket.Conn
	Server   *IMServer
	InChan   chan *wsMessage // 读队列
	OutChan  chan *wsMessage // 写队列

	Mutex     sync.Mutex
	IsClosed  bool
	CloseChan chan byte
}

func (conn *IMConnection) write(messageType int, data []byte) error {
	select {
	case conn.OutChan <- &wsMessage{messageType, data}:
	case <-conn.CloseChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (conn *IMConnection) read() (*wsMessage, error) {
	select {
	case msg := <-conn.InChan:
		return msg, nil
	case <-conn.CloseChan:
	}
	return nil, errors.New("websocket closed")
}

// 心跳发送失败,则关闭客户端连接
func (conn *IMConnection) heartbeat() {
	for {
		time.Sleep(2 * time.Second)
		hbContent := "[HEARTBEAT]heartbeat from server"
		connList := conn.Server.ConnList(conn.ID)
		if len(connList) != 0 {
			hbContent = "[ACTIVE]" + strings.Join(connList, ",")
		}
		if err := conn.write(websocket.TextMessage, []byte(hbContent)); err != nil {
			fmt.Println("heartbeat fail")
			conn.close()
			break
		}
	}
}

// 更新消息
func (conn *IMConnection) refreshMsg() {
	for {
		msg, err := conn.read()
		if err != nil {
			fmt.Println("read fail")
			break
		}
		user, msgBody, err := msg.parse()
		if err != nil {
			logrus.Fatal("message parse error:", err.Error())
			break
		}
		if len(user) == 0 || len(msgBody) == 0 {
			logrus.Warnln("nil user or msg", user, msgBody)
			break
		}
		msgBody = "[RESPONSE]" + msgBody
		conn.Server.Send(user, []byte(msgBody))
	}
}

// 协程执行ws消息发送
func (conn *IMConnection) writeTask() {
	for {
		select {
		case msg := <-conn.OutChan:
			logrus.Println("pure msg: ", string(msg.data))
			if err := conn.WsSocket.WriteMessage(msg.messageType, msg.data); err != nil {
				logrus.Fatal(err.Error())
				goto error
			}
		case <-conn.CloseChan:
			goto closed
		}
	}
error:
	conn.close()
closed:
	conn.close()
}

// 协程执行ws消息读取
func (conn *IMConnection) readTask() {
	for {
		msgType, data, err := conn.WsSocket.ReadMessage()
		if err != nil {
			goto error
		}
		req := &wsMessage{
			msgType,
			data,
		}
		select {
		case conn.InChan <- req:
		case <-conn.CloseChan:
			goto closed
		}
	}
error:
	conn.close()
closed:
	conn.close()
}

// 关闭ws连接
func (conn *IMConnection) close() {
	conn.WsSocket.Close()

	conn.Mutex.Lock()
	if !conn.IsClosed {
		conn.IsClosed = true
		close(conn.CloseChan)
	}
	conn.Mutex.Unlock()
	conn.Server.RemoveConn(conn.ID)
}
