package netway

import (
	"context"
	"io"
	"net"
	"strings"
	"time"
)

type TcpServer struct {
	listener net.Listener
	//pool []*TcpConn
}
func TcpServerNew()*TcpServer{
	tcpServer := new (TcpServer)
	//tcpServer.pool = []*TcpConn{}
	return tcpServer
}

func  (tcpServer *TcpServer)Start(){
	ipPort := mynetWay.Option.ListenIp + ":" +mynetWay.Option.Port
	listener,err :=net.Listen("tcp",ipPort)
	if err !=nil{
		mylog.Error("net.Listen tcp err")
		return
	}
	mylog.Info("startTcpServer:")
	tcpServer.listener = listener
	tcpServer.Accept()

}

func   (tcpServer *TcpServer)Shutdown( ctx context.Context){
	mylog.Warning("tcpServer Shutdown wait ctx.Done... ")
	<- ctx.Done()
	mylog.Error("tcpServer.listener.Close")
	err := tcpServer.listener.Close()
	if err != nil{
		mylog.Error("tcpServer.listener.Close err :",err)
	}
}

func (tcpServer *TcpServer)Accept( ){
	for {
		//循环接入所有客户端得到专线连接
		conn,err := tcpServer.listener.Accept()
		mylog.Info("listener.Accept new conn:")
		if err != nil{
			mylog.Error("listener.Accept err :",err.Error())
			if strings.Contains(err.Error(), "use of closed network connection") {
				mylog.Warning("TcpAccept end.")
				break
			}
			continue
		}
		tcpConn := TcpConnNew(conn)
		//myTcpServer.pool = append(myTcpServer.pool,tcpConn)
		go tcpConn.start()
	}
}
//====================================================================
type TcpConn struct {
	conn net.Conn
	MsgQueue [][]byte
	callbackCloseHandle func(code int, text string)error
}

func TcpConnNew(conn net.Conn)*TcpConn{
	mylog.Info("TcpConnNew")
	tcpConn := new (TcpConn)
	tcpConn.conn = conn
	tcpConn.callbackCloseHandle = nil
	return tcpConn
}

func  (tcpConn *TcpConn)start(){
	mylog.Info("TcpConnNew.start")
	go tcpConn.readLoop();
	time.Sleep(time.Millisecond * 500)
	mynetWay.tcpHandler(tcpConn)
}

func  (tcpConn *TcpConn)SetCloseHandler(h func(code int, text string)error) {
	tcpConn.callbackCloseHandle = h
}

func  (tcpConn *TcpConn)Close()error{
	//myTcpServer.pool[]
	tcpConn.realClose(1)
	return nil
}

func  (tcpConn *TcpConn)realClose(source int){
	if tcpConn.callbackCloseHandle != nil{
		tcpConn.callbackCloseHandle(555,"close")
	}
	mylog.Warning("realClose :",source)
	err := tcpConn.conn.Close()
	mylog.Error("tcpConn.conn.Close:",err)
}

func  (tcpConn *TcpConn)readLoop(){
	mylog.Info("new readLoop:")
	//创建消息缓冲区
	buffer := make([]byte, 1024)
	isBreak := 0
	for {
		if isBreak == 1{
			break
		}
		//读取客户端发来的消息放入缓冲区
		n,err := tcpConn.conn.Read(buffer)
		if err != nil{
			mylog.Error("conn.read buffer:",err.Error())
			if err == io.EOF{
				tcpConn.realClose(2)
				return
			}
			continue
		}
		if n == 0{
			continue
		}
		//转化为字符串输出
		clientMsg := buffer[0:n]
		mylog.Info("read msg :",n,string(clientMsg))
		//fmt.Printf("收到消息",conn.RemoteAddr(),clientMsg)
		tcpConn.MsgQueue = append(tcpConn.MsgQueue,clientMsg)
		tcpConn.conn.Write([]byte("im server"))
	}
}

func  (tcpConn *TcpConn)ReadMessage()(messageType int, p []byte, err error){
	if len(tcpConn.MsgQueue) == 0 {
		str := ""
		return messageType,[]byte(str),nil
	}
	data := tcpConn.MsgQueue[0]
	tcpConn.MsgQueue = tcpConn.MsgQueue[1:]
	return messageType,data,nil
}

func  (tcpConn *TcpConn)WriteMessage(messageType int, data []byte) error{
	tcpConn.conn.Write(data)
	return nil
}