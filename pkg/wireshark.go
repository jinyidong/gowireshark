/**
 * @Author: Administrator
 * @Description:
 * @File:  wireshark
 * @Version: 1.0.0
 * @Date: 2019/12/10 19:47
 */

package pkg

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
)

var (
	snapshotLen = int32(65535)
	promiscuous = false
	timeout     = pcap.BlockForever
	portTraffic sync.Map
)

func WireShark(deviceName string, port uint16) {
	filter := getFilter(port)
	handle, err := pcap.OpenLive(deviceName, snapshotLen, promiscuous, timeout)
	if err != nil {
		log.Error("pcap open live failed: %v", err)
		return
	}
	if err := handle.SetBPFFilter(filter); err != nil {
		fmt.Printf("set bpf filter failed: %v", err)
		return
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.NoCopy = true

	for packet := range packetSource.Packets() {
		if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
			fmt.Println("unexpected packet")
			continue
		}

		var srcIP, srcPort, dstIP, dstPort string

		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer != nil {
			ip, _ := ipLayer.(*layers.IPv4)
			srcIP = ip.SrcIP.String()
			dstIP = ip.DstIP.String()
		}

		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			srcPort = tcp.SrcPort.String()
			dstPort = tcp.DstPort.String()
		}

		log.Infof("%s:%s  ->  %s:%s", srcIP, srcPort, dstIP, dstPort)
		if !strings.Contains(srcPort, strconv.Itoa(int(port))) {
			continue
		}

		applicationLayer := packet.ApplicationLayer()
		if applicationLayer == nil {
			log.Warn("applicationLayer is nil")
			continue
		}

		key := fmt.Sprintf("%s:%s", dstIP, dstPort)
		if value, ok := portTraffic.Load(key); ok {
			if v, ok := value.(int); ok {
				portTraffic.Store(key, v+len(applicationLayer.Payload()))
				log.Infof("%s:%d", key, v+len(applicationLayer.Payload()))
			}
		} else {
			portTraffic.Store(key, len(applicationLayer.Payload()))
		}
	}
}

//定义过滤器
func getFilter(port uint16) string {
	filter := fmt.Sprintf("tcp and ((src port %v) or (dst port %v))", port, port)
	return filter
}