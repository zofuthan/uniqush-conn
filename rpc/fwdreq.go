/*
 * Copyright 2012 Nan Deng
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

import "time"

type ForwardRequest struct {
	NeverDigest   bool `json:"never-digest,omitempty"`
	DontPropagate bool `json:"dont-propagate,omitempty"`
	DontPush      bool `json:"dont-push,omitempty"`
	DontCache     bool `json:"dont-cache,omitempty"`

	DontAsk bool `json:"dont-ask-permission,omitempty"`

	Receivers       []string      `json:"receivers"`
	ReceiverService string        `json:"receiver-service"`
	TTL             time.Duration `json:"ttl"`
	MessageContainer
}
