package ipc

import "github.com/goccy/go-json"

type MsgType uint32

const (
	RunCmd MsgType = iota
	GetWorkspaces
	Subscribe
	GetOutputs
	GetTree
	GetMarks
	GetBarConfig
	GetVersion
	GetBindingModes
	GetConfig
	SendTick
	Sync
	GetBindingState
	GetInputs MsgType = 100
	GetSeats  MsgType = 101

	// used to identify async events after subscribe
	EventWorkspace       MsgType = 1000
	EventMode            MsgType = 1002
	EventWindow          MsgType = 1003
	EventBarconfigUpdate MsgType = 1004
	EventBinding         MsgType = 1005
	EventShutdown        MsgType = 1006
	EventTick            MsgType = 1007
	EventBarStateUpdate  MsgType = 1014
	EventInput           MsgType = 1015
)

type Event string

const (
	Workspace       Event = "workspace"
	Mode            Event = "mode"
	Window          Event = "window"
	BarconfigUpdate Event = "barconfig_update"
	Binding         Event = "binding"
	Shutdown        Event = "shutdown"
	Tick            Event = "tick"
	BarStateUpdate  Event = "bar_state_update"
	Input           Event = "input"
)

var IPC_HEADER = []byte("i3-ipc")
var HEADER_LEN = len(IPC_HEADER)

type Msg struct {
	MsgType    MsgType
	PayloadLen int32
	Payload    []byte
}

func NewMsg(msgType MsgType, payload []byte) *Msg {
	return &Msg{
		MsgType:    msgType,
		PayloadLen: (int32)(len(payload)),
		Payload:    payload,
	}
}

func (msg *Msg) FromJson(result interface{}) error {
	if err := json.Unmarshal(msg.Payload, result); err != nil {
		return err
	}
	return nil
}

func (msg *Msg) bytes() []byte {
	bytes := make([]byte, HEADER_LEN, HEADER_LEN+8)
	copy(bytes, IPC_HEADER)

	payloadLen := int32(len(msg.Payload))
	bytes = append(bytes, byte(payloadLen), byte(payloadLen>>8), byte(payloadLen>>16), byte(payloadLen>>24))
	bytes = append(bytes, byte(msg.MsgType), byte(msg.MsgType>>8), byte(msg.MsgType>>16), byte(msg.MsgType>>24))
	bytes = append(bytes, []byte(msg.Payload)...)

	return bytes
}
