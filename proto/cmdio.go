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

package proto

import (
	"io"
	"hash"
	"crypto/aes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/cipher"
	"code.google.com/p/snappy-go/snappy"
	"labix.org/v2/mgo/bson"
	"encoding/binary"
	"net"
)

type commandIO struct {
	writeAuth hash.Hash
	cryptWriter io.Writer

	readAuth hash.Hash
	cryptReader io.Reader

	conn net.Conn
}

func (self *commandIO) writeThenHmac(data []byte) (mac []byte, err error) {
	self.writeAuth.Reset()
	err = writen(self.cryptWriter, data)
	if err != nil {
		return
	}
	mac = self.writeAuth.Sum(nil)
	return
}

func (self *commandIO) readThenHmac(data []byte) (mac []byte, err error) {
	self.readAuth.Reset()
	n, err := io.ReadFull(self.cryptReader, data)
	if err != nil {
		return
	}
	if n != len(data) {
		err = io.EOF
		return
	}
	mac = self.readAuth.Sum(nil)
	return
}

func cmpHmac(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, c := range a {
		if c != b[i] {
			return false
		}
	}
	return true
}

func (self *commandIO) writeHmac(mac []byte) error {
	return writen(self.conn, mac)
}

func (self *commandIO) readAndCmpHmac(mac []byte) error {
	macRecved := make([]byte, self.readAuth.BlockSize())
	n, err := io.ReadFull(self.conn, macRecved)
	if err != nil {
		return err
	}
	if n != len(macRecved) {
		return ErrCorruptedData
	}
	if !cmpHmac(mac, macRecved) {
		return ErrCorruptedData
	}
	return nil
}

func (self *commandIO) readEncodedMessage(data []byte) (cmd *command, err error) {
	decoded, err := snappy.Decode(nil, data)
	if err != nil {
		return
	}
	cmd = new(command)
	err = bson.Unmarshal(decoded, cmd)
	if err != nil {
		return
	}
	return
}

func (self *commandIO) encodeCommand(cmd *command)(data []byte, err error) {
	bsonEncoded, err := bson.Marshal(cmd)
	if err != nil {
		return
	}
	data, err = snappy.Encode(nil, bsonEncoded)
	if err != nil {
		return
	}
	return
}

func (self *commandIO) WriteCommand(cmd *command) error {
	data, err := self.encodeCommand(cmd)
	if err != nil {
		return err
	}
	var cmdLen uint16
	cmdLen = uint16(len(data))
	err = binary.Write(self.conn, binary.LittleEndian, cmdLen)
	mac, err := self.writeThenHmac(data)
	if err != nil {
		return err
	}
	err = self.writeHmac(mac)
	if err != nil {
		return err
	}
	return nil
}

func (self *commandIO) ReadCommand() (cmd *command, err error) {
	var cmdLen uint16
	err = binary.Read(self.conn, binary.LittleEndian, &cmdLen)
	if err != nil {
		return
	}
	data := make([]byte, int(cmdLen))
	mac, err := self.readThenHmac(data)
	if err != nil {
		return
	}
	err = self.readAndCmpHmac(mac)
	if err != nil {
		return
	}
	cmd, err = self.readEncodedMessage(data)
	return
}

func newCommandIO(writeKey, writeAuthKey, readKey, readAuthKey []byte, conn net.Conn) *commandIO {
	ret := new(commandIO)
	ret.writeAuth = hmac.New(sha256.New, writeAuthKey)
	ret.readAuth = hmac.New(sha256.New, readAuthKey)
	ret.conn = conn

	writeBlkCipher, _ := aes.NewCipher(writeKey)
	readBlkCipher, _ := aes.NewCipher(readKey)

	// IV: 0 for all. Since we change keys for each connection, letting IV=0 won't hurt.
	writeIV := make([]byte, writeBlkCipher.BlockSize())
	readIV := make([]byte, readBlkCipher.BlockSize())

	writeStream := cipher.NewCTR(writeBlkCipher, writeIV)
	readStream := cipher.NewCTR(readBlkCipher, readIV)

	// Then for each encrypted bit,
	// it will be written to both the connection and the hmac
	// We use encrypt-then-hmac scheme.
	mwriter := io.MultiWriter(conn, ret.writeAuth)
	swriter := new(cipher.StreamWriter)
	swriter.S = writeStream
	swriter.W = mwriter
	ret.cryptWriter = swriter

	// Similarly, for each bit read from the connection,
	// it will be written to the hmac as well.
	tee := io.TeeReader(conn, ret.readAuth)
	sreader := new(cipher.StreamReader)
	sreader.S = readStream
	sreader.R = tee
	ret.cryptReader = sreader
	return ret
}

