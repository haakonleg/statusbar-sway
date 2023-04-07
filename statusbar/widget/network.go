package widget

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/godbus/dbus/v5"
)

// nf-md-wifi
const ICON_WIFI_4 = '󰤨'
const ICON_WIFI_3 = '󰤥'
const ICON_WIFI_2 = '󰤢'
const ICON_WIFI_1 = '󰤟'

// nf-md-ethernet
const ICON_ETHERNET = '󰈀'

type netType int

const (
	typeEthernet netType = iota
	typeWifi
	typeUnknown
)

type networkManagerInfoResult struct {
	connections       []*netConnection
	primaryConnection *netConnection
}

type ethernetData struct {
	speed uint32
}

type wifiData struct {
	ssid          string
	bitrate       uint32
	signalQuality uint8
}

type netConnection struct {
	ifName  string
	netType netType
	rxBytes uint64
	txBytes uint64
	data    any
}

type Network struct {
	*Widget
	dbus              *dbus.Conn
	connections       []*netConnection
	primaryConnection *netConnection

	nmInfoChannel   chan *networkManagerInfoResult
	nmSignalChannel chan *dbus.Signal
}

func NewNetworkWidget() *Widget {
	return newWidget("network", -1, func(widget *Widget) impl {
		return &Network{
			Widget:          widget,
			nmInfoChannel:   make(chan *networkManagerInfoResult, 1),
			nmSignalChannel: make(chan *dbus.Signal, 10),
		}
	})
}

func (n *Network) setup() {
	// setup dbus
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatalf("failed to connect to dbus: %s", err.Error())
	}

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/NetworkManager"),
	); err != nil {
		log.Fatalf(err.Error())
	}

	// subscribe to dbus signals
	conn.Signal(n.nmSignalChannel)

	n.dbus = conn
}

func (n *Network) close() {
	n.dbus.Close()
}

// listen for dbus signals and info update
func (n *Network) run() {
	go n.updateNetworkManagerInfo()

	infoUpdate := time.NewTicker(30 * time.Second)
	for {
		select {
		case info := <-n.nmInfoChannel:
			n.connections = info.connections
			n.primaryConnection = info.primaryConnection

			// trigger immediate update
			n.sendUpdate()

		case sig := <-n.nmSignalChannel:
			log.Printf("signal: %+v", sig)

		case <-infoUpdate.C:
			log.Println("updating network manager info")
			go n.updateNetworkManagerInfo()
		}
	}
}

func (n *Network) update(block *block) {
	if n.connections != nil && len(n.connections) > 0 {
		primary := n.primaryConnection

		if primary.netType == typeWifi {
			info := primary.data.(*wifiData)
			block.FullText = fmt.Sprintf("%c %s (%s)", ICON_WIFI_2, primary.ifName, info.ssid)
		} else if primary.netType == typeEthernet {
			block.FullText = fmt.Sprintf("%c %s", ICON_ETHERNET, primary.ifName)
		}
	} else {
		block.FullText = "No connection"
	}
}

func (c *Network) onClick(x int, y int, btn int) {}

func (n *Network) updateNetworkManagerInfo() {
	result := &networkManagerInfoResult{}

	var props map[string]interface{}
	if err := n.nmDbusCall(&props, "/org/freedesktop/NetworkManager", ""); err != nil {
		return
	}

	primaryConnection := props["PrimaryConnection"].(dbus.ObjectPath)
	devices := props["Devices"].([]dbus.ObjectPath)

	// get active connections
	connections := make([]*netConnection, 0)
	for _, device := range devices {
		if err := n.nmDbusCall(&props, device, ".Device"); err != nil {
			continue
		}

		activeConnection := props["ActiveConnection"].(dbus.ObjectPath)
		if activeConnection == "/" {
			continue
		}

		connection := &netConnection{
			ifName:  props["Interface"].(string),
			netType: typeUnknown,
		}

		if activeConnection == primaryConnection {
			result.primaryConnection = connection
		}

		typeNum := props["DeviceType"].(uint32)
		if typeNum == 1 {
			// NM_DEVICE_TYPE_ETHERNET
			connection.netType = typeEthernet
			ethernetInfo := &ethernetData{}

			if err := n.nmDbusCall(&props, device, ".Device.Wired"); err == nil {
				ethernetInfo.speed = props["Speed"].(uint32)
			}

			connection.data = ethernetInfo
		} else if typeNum == 2 {
			// NM_DEVICE_TYPE_WIFI
			connection.netType = typeWifi
			wifiInfo := &wifiData{}
			var accessPoint dbus.ObjectPath

			if err := n.nmDbusCall(&props, device, ".Device.Wireless"); err == nil {

				wifiInfo.bitrate = props["Bitrate"].(uint32)
				accessPoint = props["ActiveAccessPoint"].(dbus.ObjectPath)
			}

			// get access point
			if accessPoint.IsValid() {
				if err := n.nmDbusCall(&props, accessPoint, ".AccessPoint"); err == nil {
					ssid := props["Ssid"].([]uint8)
					wifiInfo.ssid = string(ssid)
					wifiInfo.signalQuality = props["Strength"].(uint8)
				}
			}

			connection.data = wifiInfo
		}

		// get statistics
		if err := n.nmDbusCall(&props, device, ".Device.Statistics"); err == nil {
			connection.rxBytes = props["RxBytes"].(uint64)
			connection.txBytes = props["TxBytes"].(uint64)
		}

		connections = append(connections, connection)
	}

	result.connections = connections
	n.nmInfoChannel <- result
}

func (n *Network) nmDbusCall(result interface{}, object dbus.ObjectPath, ifname string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	resultCh := make(chan *dbus.Call, 1)
	bus := n.dbus.Object("org.freedesktop.NetworkManager", object)
	bus.GoWithContext(ctx, "org.freedesktop.DBus.Properties.GetAll", 0, resultCh, "org.freedesktop.NetworkManager"+ifname)

	select {
	case <-ctx.Done():
		log.Printf("failed to call NetworkManager: timeout")
		return errors.New("timeout")

	case call := <-resultCh:
		if call.Err != nil {
			log.Printf("failed to call NetworkManager: %s", call.Err.Error())
			return call.Err
		} else {
			call.Store(result)
			return nil
		}
	}
}

/*
func (n *Network) readNetworkData() {
	n.numInterfaces = 0

	// get interfaces
	n.netDevFile.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(n.netDevFile)

	scanner.Scan()
	scanner.Scan()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip loopback interface
		if strings.HasPrefix(line, "lo") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) > 1 {
			if len(n.netInterfaces) < n.numInterfaces+1 {
				n.netInterfaces = append(n.netInterfaces, &netInterface{})
			}

			netInterface := n.netInterfaces[n.numInterfaces]
			netInterface.name = fields[0][:len(fields[0])-1]
			netInterface.isWireless = false
			netInterface.signalQuality = 0
			n.numInterfaces++
		}
	}

	n.netWirelessFile.Seek(0, io.SeekStart)
	scanner = bufio.NewScanner(n.netWirelessFile)

	scanner.Scan()
	scanner.Scan()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		for i := 0; i < n.numInterfaces; i++ {
			netInterface := n.netInterfaces[i]
			if !strings.HasPrefix(line, netInterface.name) {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) > 1 {
				netInterface.isWireless = true

				signalQuality := fields[4][:len(fields[4])-1]
				netInterface.signalQuality, _ = strconv.Atoi(signalQuality)
			}
		}
	}

	n.netRouteFile.Seek(0, io.SeekStart)
	scanner = bufio.NewScanner(n.netRouteFile)

	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()

		for i := 0; i < n.numInterfaces; i++ {
			netInterface := n.netInterfaces[i]
			if !strings.HasPrefix(line, netInterface.name) {
				continue
			}

			fields := strings.Fields(line)
			destination := fields[1]

			if destination == "00000000" {
			}
		}
	}
}
*/
