package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"bytes"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type session struct {
	ws      *websocket.Conn
	rl      *readline.Instance
	errChan chan error
}

func connect(url, origin string, rlConf *readline.Config) error {
	headers := make(http.Header)
	headers.Add("Origin", origin)

	ws, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		return err
	}

	rl, err := readline.NewEx(rlConf)
	if err != nil {
		return err
	}
	defer rl.Close()

	sess := &session{
		ws:      ws,
		rl:      rl,
		errChan: make(chan error),
	}

	go sess.readConsole()
	go sess.readWebsocket()

	return <-sess.errChan
}

func (s *session) readConsole() {
	for {
		line, err := s.rl.Readline()
		if err != nil {
			s.errChan <- err
			return
		}

		err = s.ws.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			s.errChan <- err
			return
		}
	}
}

func bytesToFormattedHex(bytes []byte) string {
	text := hex.EncodeToString(bytes)
	return regexp.MustCompile("(..)").ReplaceAllString(text, "$1 ")
}

func (s *session) readWebsocket() {
	rxSprintf := color.New(color.FgGreen).SprintfFunc()

	for {
		msgType, buf, err := s.ws.ReadMessage()
		if err != nil {
			s.errChan <- err
			return
		}

		var text string
		switch msgType {
		case websocket.TextMessage:
			text = string(buf)
		case websocket.BinaryMessage:
			text = bytesToFormattedHex(buf)
		default:
			s.errChan <- fmt.Errorf("unknown websocket frame type: %d", msgType)
			return
		}

		heartbeat := []byte("{\"type\":\"heartbeat\"}")
		if bytes.Compare([]byte(text), heartbeat) != 0 {
			fmt.Fprint(s.rl.Stdout(), rxSprintf("< %s\n", text))
			size := len([]byte(text))
			fmt.Fprint(s.rl.Stdout(), rxSprintf("< JSON payload size: %d bytes\n", size))
			fmt.Fprint(s.rl.Stdout(), rxSprintf("<\n"))
		}
	}
}
