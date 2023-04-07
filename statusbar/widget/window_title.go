package widget

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/haakonleg/statusbar-sway/ipc"
)

type WindowInfo struct {
	name string
	pid  int
}

type WindowTitle struct {
	*Widget
	ipcClient *ipc.SwayIpcClient

	// currently focused window
	focusedWindow     *WindowInfo
	processNamesByPid map[int]string
}

func NewWindowTitleWidget() *Widget {
	return newWidget("window_title", -1, func(widget *Widget) impl {
		return &WindowTitle{
			Widget:            widget,
			processNamesByPid: make(map[int]string),
		}
	})
}

func (w *WindowTitle) setup() {
	ipcClient, err := ipc.Connect()
	if err != nil {
		log.Fatalf("failed to connect to sway ipc protocol: %s", err.Error())
	}
	w.ipcClient = ipcClient
}

func (w *WindowTitle) close() {
	w.ipcClient.Close()
}

func (w *WindowTitle) run() {
	// send GetTree message for initial update
	err := w.ipcClient.Send(ipc.NewMsg(ipc.GetTree, []byte{}))
	if err != nil {
		log.Fatalf("failed to send sway ipc command: %s", err.Error())
	}

	// subscribe to Window event
	if err := w.ipcClient.SubscribeEvent(ipc.Window, ipc.Workspace); err != nil {
		log.Fatalf("failed to subscribe to event: %s", err.Error())
	}

	// handle replies and events
	for {
		msg := <-w.ipcClient.MsgQueue

		if msg.MsgType == ipc.GetTree {
			var response map[string]interface{}
			msg.FromJson(&response)

			w.focusedWindow = findFocusedWindow(response)
			w.sendUpdate()

		} else if msg.MsgType == ipc.EventWindow {
			var response map[string]interface{}
			msg.FromJson(&response)

			change := response["change"].(string)
			if change == "focus" || change == "title" {
				container := response["container"].(map[string]interface{})

				name, hasName := container["name"].(string)
				pid, hasPid := container["pid"].(float64)
				if hasName && hasPid {
					w.focusedWindow = &WindowInfo{name: name, pid: (int)(pid)}
					w.sendUpdate()
				}
			} else if change == "close" {
				w.ipcClient.Send(ipc.NewMsg(ipc.GetTree, []byte{}))
			}

		} else if msg.MsgType == ipc.EventWorkspace {
			var response map[string]interface{}
			msg.FromJson(&response)

			change := response["change"].(string)
			if change == "focus" {
				w.ipcClient.Send(ipc.NewMsg(ipc.GetTree, []byte{}))
			}
		}

	}
}

func (w *WindowTitle) update(block *block) {
	if w.focusedWindow != nil {
		// cache process name to avoid opening reading file every time
		procName, cached := w.processNamesByPid[w.focusedWindow.pid]
		if !cached {
			procName = getProcessNameFromPID(w.focusedWindow.pid)
			w.processNamesByPid[w.focusedWindow.pid] = procName
		}

		block.FullText = fmt.Sprintf("%s - %.50s", procName, w.focusedWindow.name)
	} else {
		block.FullText = ""
	}
}

func (c *WindowTitle) onClick(x int, y int, btn int) {}

// parses the tree from command GET_TREE, and returns the focused window
func findFocusedWindow(node map[string]interface{}) *WindowInfo {
	for key, value := range node {
		if key == "focused" {
			isFocused := value.(bool)

			if isFocused {
				name, hasName := node["name"].(string)
				pid, hasPid := node["pid"].(float64)

				if hasName && hasPid {
					return &WindowInfo{name: name, pid: (int)(pid)}
				}
			}
		}

		if key == "nodes" {
			childNodes := value.([]interface{})

			for _, childNode := range childNodes {
				child := childNode.(map[string]interface{})
				result := findFocusedWindow(child)

				if result != nil {
					return result
				}
			}
		}
	}

	return nil
}

func getProcessNameFromPID(pid int) string {
	pidStr := strconv.Itoa(pid)
	cmdline, err := os.Open("/proc/" + pidStr + "/cmdline")
	if err != nil {
		log.Printf("failed to read cmdline for pid %d: %s", pid, err.Error())
		return pidStr
	}
	defer cmdline.Close()

	reader := bufio.NewReader(cmdline)
	procName, err := reader.ReadSlice(' ')
	if err != nil && err != io.EOF {
		return pidStr
	}
	procName = procName[:len(procName)-1]

	parts := strings.Split(string(procName), "/")
	return parts[len(parts)-1]
}
