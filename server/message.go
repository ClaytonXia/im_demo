package server

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/sirupsen/logrus"
)

type wsMessage struct {
	messageType int
	data        []byte
}

func (m *wsMessage) parse() (user, msg string, err error) {
	msgData := m.data
	if len(msgData) == 0 {
		logrus.Warnln("nil message")
		return
	}
	logrus.Println(string(msgData))

	msgReader := bufio.NewReader(bytes.NewBuffer(msgData))
	user, err = msgReader.ReadString(' ')
	if err != nil {
		logrus.Error(err.Error())
		return
	}
	user = strings.TrimSpace(user)
	msg, err = msgReader.ReadString('\n')
	if err != nil {
		logrus.Error(err.Error())
		return
	}

	return
}
