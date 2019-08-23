package comm

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"syncing/gproto"

	"github.com/golang/protobuf/proto"
)

type Connection struct {
	rBuf *bufio.Reader
	wBuf *bufio.Writer
}

func NewConn(rb *bufio.Reader, wb *bufio.Writer) *Connection {
	conn := &Connection{}
	conn.Init(rb, wb)
	return conn
}

func (self *Connection) Init(rb *bufio.Reader, wb *bufio.Writer) {
	self.rBuf = rb
	self.wBuf = wb
}

func (self *Connection) Send(id int8, data []byte) (int, error) {
	var header gproto.Header
	header.Id = id
	header.Len = int32(len(data))

	err := binary.Write(self.wBuf, binary.LittleEndian, header)
	if err != nil {
		return 0, err
	}

	n, err := self.wBuf.Write(data)
	if err != nil {
		return n, err
	}
	self.wBuf.Flush()
	return n, nil
}

func (self *Connection) Recv() (int8, interface{}, error) {
	header := gproto.Header{}
	err := binary.Read(self.rBuf, binary.LittleEndian, &header)
	if err != nil {
		return 0, nil, err
	}
	cmd := header.Id
	length := int(header.Len)

	var dataBuf []byte
	if length > 0 {
		dataBuf = make([]byte, length)
		_, err = io.ReadFull(self.rBuf, dataBuf)
		if err != nil {
			return 0, nil, err
		}
	}

	switch cmd {
	case gproto.MSG_A_INITPARAM:
		var initParam gproto.InitParam
		err := proto.Unmarshal(dataBuf, &initParam)
		return cmd, &initParam, err

	case gproto.MSG_A_DIR_INFO:
		var st gproto.DirStruct
		err := proto.Unmarshal(dataBuf, &st)
		return cmd, &st, err

	case gproto.MSG_A_PATCHLIST:
		var patchList gproto.PatchList
		err := proto.Unmarshal(dataBuf, &patchList)
		return cmd, &patchList, err

	case gproto.MSG_B_SUMLIST:
		var fileSumList gproto.FileSumList
		err = proto.Unmarshal(dataBuf, &fileSumList)
		return cmd, &fileSumList, err

	case gproto.MSG_A_END:
		return cmd, nil, nil

	case gproto.MSG_B_END:
		var syncResult gproto.SyncResult
		err = proto.Unmarshal(dataBuf, &syncResult)
		return cmd, &syncResult, err
	}

	return 0, nil, errors.New("not hit")
}

func (self *Connection) Flush() {
	self.wBuf.Flush()
}
