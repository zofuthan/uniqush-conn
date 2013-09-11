/*
 * Copyright 2013 Nan Deng
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package msgcenter

import (
	"fmt"
	"github.com/uniqush/uniqush-conn/config"
	"github.com/uniqush/uniqush-conn/proto/server"
	"github.com/uniqush/uniqush-conn/rpc"
	"io"
	"strings"
)

type serviceCenter struct {
	config     *config.ServiceConfig
	fwdChan    chan<- *rpc.ForwardRequest
	subReqChan chan<- *server.SubscribeRequest
	conns      connMap
}

func (self *serviceCenter) serveConn(conn server.Conn) {
	var reason error
	defer func() {
		self.conns.DelConn(conn)
		self.config.OnLogout(conn, reason)
		conn.Close()
	}()
	for {
		msg, err := conn.ReceiveMessage()
		if err != nil {
			if err != io.EOF {
				self.config.OnError(conn, err)
				reason = err
			}
		}
		if msg != nil {
			self.config.OnMessage(conn, msg)
		}
	}
}

func (self *serviceCenter) NewConn(conn server.Conn) {
	if conn == nil {
		//self.config.OnError(conn, fmt.Errorf("Nil conn")
		return
	}
	usr := conn.Username()
	if len(usr) == 0 || strings.Contains(usr, ":") || strings.Contains(usr, "\n") {
		self.config.OnError(conn, fmt.Errorf("invalid username"))
		conn.Close()
		return
	}
	conn.SetMessageCache(self.config.Cache())
	conn.SetForwardRequestChannel(self.fwdChan)
	conn.SetSubscribeRequestChan(self.subReqChan)
	err := self.conns.AddConn(conn)
	if err != nil {
		self.config.OnError(conn, err)
		conn.Close()
		return
	}

	go self.serveConn(conn)
	return
}

func (self *serviceCenter) Send(req *rpc.SendRequest) *rpc.Result {
	ret := new(rpc.Result)

	if req == nil {
		ret.Error = fmt.Errorf("invalid request")
		return ret
	}
	if req.Message == nil || req.Message.IsEmpty() {
		ret.Error = fmt.Errorf("invalid request: empty message")
		return ret
	}
	if req.Receiver == "" {
		ret.Error = fmt.Errorf("invalid request: no receiver")
		return ret
	}

	conns := self.conns.GetConn(req.Receiver)
	var mid string
	msg := req.Message
	mc := &rpc.MessageContainer{
		Sender:        "",
		SenderService: "",
		Message:       msg,
	}
	mid, ret.Error = self.config.CacheMessage(req.Receiver, mc, req.TTL)
	if ret.Error != nil {
		return ret
	}

	n := 0

	for _, minc := range conns {
		if conn, ok := minc.(server.Conn); ok {
			err := conn.SendMessage(msg, mid, nil)
			ret.Append(conn, err)
			if err != nil {
				conn.Close()
			} else {
				n++
			}
		} else {
			self.conns.DelConn(minc)
		}
	}

	if n == 0 && !req.DontPush {
		self.config.Push(req.Receiver, "", "", req.PushInfo, mid, msg.Size())
	}
	return ret
}

func (self *serviceCenter) Forward(req *rpc.ForwardRequest, dontAsk bool) *rpc.Result {
	ret := new(rpc.Result)

	if req == nil {
		ret.Error = fmt.Errorf("invalid request")
		return ret
	}
	if req.Message == nil || req.Message.IsEmpty() {
		ret.Error = fmt.Errorf("invalid request: empty message")
		return ret
	}
	if req.Receiver == "" {
		ret.Error = fmt.Errorf("invalid request: no receiver")
		return ret
	}

	conns := self.conns.GetConn(req.Receiver)
	var mid string
	msg := req.Message
	mc := &req.MessageContainer

	var pushInfo map[string]string
	var shouldForward bool
	shouldPush := !req.DontPush

	if !dontAsk {
		// We need to ask for permission to forward this message.
		// This means the forward request is generated directly from a user,
		// not from a uniqush-conn node in a cluster.

		mc.Id = ""
		shouldForward, shouldPush, pushInfo = self.config.ShouldForward(req)

		if !shouldForward {
			return nil
		}
	}

	mid, ret.Error = self.config.CacheMessage(req.Receiver, mc, req.TTL)
	if ret.Error != nil {
		return ret
	}

	n := 0

	for _, minc := range conns {
		if conn, ok := minc.(server.Conn); ok {
			err := conn.SendMessage(msg, mid, nil)
			ret.Append(conn, err)
			if err != nil {
				conn.Close()
			} else {
				n++
			}
		} else {
			self.conns.DelConn(minc)
		}
	}

	if n == 0 && shouldPush {
		self.config.Push(req.Receiver, req.SenderService, req.Sender, pushInfo, mid, msg.Size())
	}
	return ret
}

func newServiceCenter(conf *config.ServiceConfig, fwdChan chan<- *rpc.ForwardRequest, subReqChan chan<- *server.SubscribeRequest) *serviceCenter {
	if conf == nil || fwdChan == nil || subReqChan == nil {
		return nil
	}
	ret := new(serviceCenter)
	ret.config = conf
	ret.conns = newTreeBasedConnMap(conf.MaxNrConns, conf.MaxNrUsers, conf.MaxNrConnsPerUser)
	ret.fwdChan = fwdChan
	ret.subReqChan = subReqChan
	return ret
}
