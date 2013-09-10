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

package rpc

import "net"

type Result struct {
	CallID  string        `json:"call-id"`
	Error   error         `json:"error,omitempty"`
	Results []*ConnResult `json:"results"`
}

type connDescriptor interface {
	RemoteAddr() net.Addr
	Service() string
	Username() string
	UniqId() string
	Visible() bool
}

func (self *Result) Append(c connDescriptor, err error) {
	if self == nil {
		return
	}
	if self.Results == nil {
		self.Results = make([]*ConnResult, 0, 10)
	}
	r := new(ConnResult)
	r.ConnId = c.UniqId()
	r.Error = err
	r.Visible = c.Visible()
	self.Results = append(self.Results, r)
}

type ConnResult struct {
	ConnId  string `json:"conn-id"`
	Error   error  `json:"error,omitempty"`
	Visible bool   `json:"visible"`
}
