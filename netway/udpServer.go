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
	msg ,err := mynetWay.parserContentProtocol( data)
	if err != nil{
		mylog.Error("parserContentProtocol err",err)
	}
	playerId ,ok := mynetWay.PlayerManager.SidMapPid[msg.SessionId]
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

	wsConn,_ := connManager.getConnPoolById(playerId)
	wsConn.UdpConn = true
	mynetWay.Router(msg,wsConn)
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

//====================================================================
//type TcpConn struct {
//	conn net.UDPConn
//	MsgQueue [][]byte
//	callbackCloseHandle func(code int, text string)error
//}
//
//func TcpConnNew(conn net.Conn)*TcpConn{
//	mylog.Info("TcpConnNew")
//	tcpConn := new (TcpConn)
//	tcpConn.conn = conn
//	tcpConn.callbackCloseHandle = nil
//	return tcpConn
//}
//
//func  (tcpConn *TcpConn)start(){
//	mylog.Info("TcpConnNew.start")
//	go tcpConn.readLoop();
//	time.Sleep(time.Millisecond * 500)
//	mynetWay.tcpHandler(tcpConn)
//}
//
//func  (tcpConn *TcpConn)SetCloseHandler(h func(code int, text string)error) {
//	tcpConn.callbackCloseHandle = h
//}
//
//func  (tcpConn *TcpConn)Close()error{
//	//myTcpServer.pool[]
//	tcpConn.realClose(1)
//	return nil
//}
//
//func  (tcpConn *TcpConn)realClose(source int){
//	if tcpConn.callbackCloseHandle != nil{
//		tcpConn.callbackCloseHandle(555,"close")
//	}
//	mylog.Warning("realClose :",source)
//	err := tcpConn.conn.Close()
//	mylog.Error("tcpConn.conn.Close:",err)
//}
//
//func  (tcpConn *TcpConn)readLoop(){
//	mylog.Info("new readLoop:")
//	//创建消息缓冲区
//	buffer := make([]byte, 1024)
//	isBreak := 0
//	for {
//		if isBreak == 1{
//			break
//		}
//		//读取客户端发来的消息放入缓冲区
//		n,err := tcpConn.conn.Read(buffer)
//		if err != nil{
//			mylog.Error("conn.read buffer:",err.Error())
//			if err == io.EOF{
//				tcpConn.realClose(2)
//				return
//			}
//			continue
//		}
//		if n == 0{
//			continue
//		}
//		//转化为字符串输出
//		clientMsg := buffer[0:n]
//		mylog.Info("read msg :",n,string(clientMsg))
//		//fmt.Printf("收到消息",conn.RemoteAddr(),clientMsg)
//		tcpConn.MsgQueue = append(tcpConn.MsgQueue,clientMsg)
//		tcpConn.conn.Write([]byte("im server"))
//	}
//}
//
//func  (tcpConn *TcpConn)ReadMessage()(messageType int, p []byte, err error){
//	if len(tcpConn.MsgQueue) == 0 {
//		str := ""
//		return messageType,[]byte(str),nil
//	}
//	data := tcpConn.MsgQueue[0]
//	tcpConn.MsgQueue = tcpConn.MsgQueue[1:]
//	return messageType,data,nil
//}
//
//func  (tcpConn *TcpConn)WriteMessage(messageType int, data []byte) error{
//	tcpConn.conn.Write(data)
//	return nil
//}