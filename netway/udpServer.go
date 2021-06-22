package netway

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"zlib"
)
type UdpSessionPlayerConn struct {
	Ip string
	Port int
	SessionId string
	PlayerId int32
	Atime 	int
}

type UdpServer struct {
	netWayOption NetWayOption
	UdpSessionPlayerConnPool map[string]*UdpSessionPlayerConn
	//listener *net.UDPConn
	//pool []*TcpConn
}
func UdpServerNew(netWayOption NetWayOption,mylogP *zlib.Log)*UdpServer{
	mylog = mylogP
	udpServer := new (UdpServer)
	//tcpServer.pool = []*TcpConn{}
	udpServer.netWayOption = netWayOption
	udpServer.UdpSessionPlayerConnPool = make(map[string]*UdpSessionPlayerConn)
	return udpServer
}

func  (udpServer *UdpServer)Start(){
	//ipPort := mynetWay.Option.ListenIp + ":" +mynetWay.Option.UdpPort
	UdpPort,_ := strconv.Atoi(udpServer.netWayOption.UdpPort)
	udpConn,err := net.ListenUDP("udp",&net.UDPAddr{
		//IP: net.IPv4(0,0,0,0),
		IP: net.IPv4(127, 0, 0, 1),
		Port: UdpPort,
	})
	if err !=nil{
		mylog.Error("net.ListenUDP tcp err")
		return
	}
	mylog.Info("start ListenUDP and loop read...",UdpPort)
	for {
		var data [1024]byte
		n,addr,err := udpConn.ReadFromUDP(data[:])
		mylog.Info("have a new msg ",n,addr,err)
		if err != nil{
			mylog.Error("udpConn.ReadFromUDP:",n,addr,err)
			break
		}

		if n == 0{
			mylog.Error("udpConn.ReadFromUDP n = 0" )
			continue
		}

		readlData := []byte{}
		for k,v := range data{
			if k > n {
				readlData = append(readlData,v)
			}
		}

		udpServer.processOneMsg(string(readlData),addr)
	}
}
func  (udpServer *UdpServer)processOneMsg(data string,addr *net.UDPAddr){
	msg ,err := myProtocolManager.parserContentProtocol( data)
	if err != nil{
		mylog.Error("parserContentProtocol err",err)
	}
	playerId ,ok := myPlayerManager.SidMapPid[msg.SessionId]
	if !ok{
		mylog.Error("mynetWay.PlayerManager.SidMapPid is empty")
		return
	}
	myUdpSessionPlayerConn,ok := udpServer.UdpSessionPlayerConnPool[msg.SessionId]
	if !ok {
		udpSessionPlayerConn := UdpSessionPlayerConn{
			Ip: string(addr.IP),
			Port: addr.Port,
			SessionId: msg.SessionId,
			PlayerId: playerId,
			Atime: zlib.GetNowTimeSecondToInt(),
		}
		udpServer.UdpSessionPlayerConnPool[msg.SessionId] = &udpSessionPlayerConn
	}else{
		myUdpSessionPlayerConn.Ip = string(addr.IP)
		myUdpSessionPlayerConn.Port= addr.Port
	}

	conn,_ := connManager.getConnPoolById(playerId)
	conn.UdpConn = true
	myNetWay.Router(msg,conn)
}

func  (udpServer *UdpServer)StartClient(){

	UdpPort,_ := strconv.Atoi(udpServer.netWayOption.UdpPort)
	// 创建连接
	socket, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: UdpPort,
	})
	if err !=nil{
		mylog.Error("net.ListenUDP tcp err")
		return
	}
	mylog.Info("StartClient ListenUDP and loop write...",UdpPort)
	defer socket.Close()
	// 发送数据
	senddata := []byte("hello server!")
	n, err := socket.Write(senddata)
	mylog.Info("socket.Write:",string(senddata),n,err)
	if err != nil {
		fmt.Println("发送数据失败!", err)
		return
	}

	for  {
		
	}
}


func   (udpServer *UdpServer)Shutdown( ctx context.Context){
	//mylog.Warning("tcpServer Shutdown wait ctx.Done... ")
	//<- ctx.Done()
	//mylog.Error("tcpServer.listener.Close")
	//err := tcpServer.listener.Close()
	//if err != nil{
	//	mylog.Error("tcpServer.listener.Close err :",err)
	//}
}
