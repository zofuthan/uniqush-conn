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

package msgcache

import (
	"github.com/uniqush/uniqush-conn/proto"
	"time"
)

type MessageInfo struct {
	Id         string         `json:"id"`
	Message    *proto.Message `json:"msg"`
	Sender     string         `json:"sender,omitempty"`
	SrcService string         `json:"srcSrv,omitempty"`
}

type Cache interface {
	CacheMessage(service, username string, msg *MessageInfo, ttl time.Duration) (id string, err error)
	// XXX Is there any better way to support retrieve all feature?
	Get(service, username, id string) (msg *MessageInfo, err error)
	GetCachedMessages(service, username string, excludes ...string) (msgs []*MessageInfo, err error)
}
