package main

import (
	"frame_sync/netway"
	"zlib"
)

//testSwitchClientServer(cmsArg["ClientServer"],newNetWayOption)
//测试使用，开始TCP/UDP client端
func testSwitchClientServer(clientServer string,newNetWayOption netway.NetWayOption ){
	switch clientServer {
	case "client":
		netway.StartTcpClient(newNetWayOption,mainlog)
		cc := make(chan int)
		<- cc
		zlib.PanicPrint(1111111111)
	case "udpClient":
		udpServer :=  netway.UdpServerNew(newNetWayOption,mainlog)
		udpServer.StartClient()
		cc := make(chan int)
		<- cc
		zlib.PanicPrint(22222)
	case "udpServer":
		udpServer :=  netway.UdpServerNew(newNetWayOption,mainlog)
		udpServer.Start()
		cc := make(chan int)
		<- cc
		zlib.PanicPrint(33333)
	}
}