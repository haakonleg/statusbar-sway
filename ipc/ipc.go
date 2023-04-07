package ipc

import (
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-json"
)

type SwayIpcClient struct {
	conn     *net.UnixConn
	MsgQueue chan *Msg
}

func Connect() (*SwayIpcClient, error) {
	addr, err := getSwaySockAddr()
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}

	client := &SwayIpcClient{
		conn:     conn,
		MsgQueue: make(chan *Msg, 10),
	}

	go client.readMsg()
	return client, nil
}

func (s *SwayIpcClient) Close() {
	s.conn.Close()
}

// Send sends a sway ipc message over the socket
func (s *SwayIpcClient) Send(msg *Msg) error {
	if err := s.writeMsg(msg); err != nil {
		return err
	}
	return nil
}

// SubscribeEvent subscribes to an event type in the sway ipc protocol
// events can be received by listening to the eventQueue channel
func (s *SwayIpcClient) SubscribeEvent(events ...Event) error {
	payload, err := json.Marshal(events)
	if err != nil {
		return err
	}

	if err = s.writeMsg(NewMsg(Subscribe, payload)); err != nil {
		return err
	}

	return nil
}

func (s *SwayIpcClient) writeMsg(msg *Msg) error {
	bytes := msg.bytes()

	// todo: test loop
	writeLen := 0
	for writeLen != len(bytes) {
		if n, err := s.conn.Write(bytes[writeLen:]); err != nil {
			return err
		} else {
			writeLen += n
		}
	}

	log.Printf("wrote msg: %+v", msg)
	return nil
}

// readMsg continuously reads from the socket and consumes messages
func (s *SwayIpcClient) readMsg() {
	header := make([]byte, HEADER_LEN+8)

	for {
		var msgType uint32 = 0
		var payloadLen uint32 = 0
		var payload []byte = nil

		// read header
		for payload == nil {
			if _, err := io.ReadFull(s.conn, header); err == io.EOF {
				// fixme: reconnect
				continue
			} else if err != nil {
				log.Printf("encountered error during read: %s", err.Error())
				continue
			}

			if !bytes.Equal(header[:HEADER_LEN], IPC_HEADER) {
				log.Printf("did not receive expected magic string in reply")
				continue
			}

			payloadLen = bytesToInt32(header[HEADER_LEN : HEADER_LEN+4])
			if payloadLen == 0 {
				break
			}

			msgType = bytesToInt32(header[HEADER_LEN+4 : HEADER_LEN+8])
			// check if event
			if (msgType >> 31) == 1 {
				msgType = (msgType & 0x7F) + 1000
			}

			payload = make([]byte, payloadLen)
		}

		if _, err := io.ReadFull(s.conn, payload); err == io.EOF {
			// fixme: reconnect
			continue
		} else if err != nil {
			log.Printf("encountered error during read: %s", err.Error())
			continue
		}

		msg := NewMsg(MsgType(msgType), payload)
		log.Printf("received msg: %d", msg.MsgType)

		select {
		case s.MsgQueue <- msg:
		default:
			// queue is full, discard first
			<-s.MsgQueue
			s.MsgQueue <- msg
		}
	}
}

func bytesToInt32(bytes []byte) uint32 {
	var val uint32
	val |= uint32(bytes[0])
	val |= uint32(bytes[1]) << 8
	val |= uint32(bytes[2]) << 16
	val |= uint32(bytes[3]) << 24
	return val
}

func getSwaySockAddr() (*net.UnixAddr, error) {
	swaySock := os.Getenv("SWAYSOCK")

	// if env variable didn't work for whatever reason
	if len(swaySock) == 0 {
		if stdout, err := exec.Command("sway", "--get-socketpath").Output(); err != nil {
			return nil, err
		} else {
			swaySock = strings.TrimSpace(string(stdout))
		}
	}

	return net.ResolveUnixAddr("unix", swaySock)
}
