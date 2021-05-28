package netway

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"zlib"
)

type UdpServer struct {
	netWayOption NetWayOption

	//listener *net.UDPConn
	//pool []*TcpConn
}
func UdpServerNew(netWayOption NetWayOption,mylogP *zlib.Log)*UdpServer{
	mylog = mylogP
	udpServer := new (UdpServer)
	//tcpServer.pool = []*TcpConn{}
	udpServer.netWayOption = netWayOption

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
			log.Printf("Read from udp server:%s failed,err:%s",addr,err)
			break
		}
		go func() {
			// 返回数据
			fmt.Printf("Addr:%s,data:%v count:%d \n",addr,string(data[:n]),n)
			_,err := udpConn.WriteToUDP([]byte("msg recived."),addr)
			if err != nil{
				fmt.Println("write to udp server failed,err:",err)
			}
		}()
	}
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