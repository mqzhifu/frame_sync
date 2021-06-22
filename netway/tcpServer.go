package netway

import (
	"context"
	"io"
	"net"
	"strings"
	"time"
	"zlib"
)
type TcpServer struct {
	listener net.Listener
	OutCxt   context.Context
	Ip 		string
	Port 	string
}
func NewTcpServer(outCtx context.Context,ip string,port string)*TcpServer{
	mylog.Info("NewTcpServer instance:")
	tcpServer := new (TcpServer)
	tcpServer.OutCxt = outCtx
	tcpServer.Ip = ip
	tcpServer.Port = port
	//tcpServer.pool = []*TcpConn{}
	//go tcpServer.ListeningClose()
	return tcpServer
}

//func (tcpServer *TcpServer)ListeningClose(){
	//<- tcpServer.OutCxt.Done()
	//tcpServer.Shutdown()
//}
//outCtx:这里没用上，因为accept是阻塞的模式，只能用另外的方式close
func  (tcpServer *TcpServer)Start(){
	ipPort := tcpServer.Ip + ":" + tcpServer.Port
	mylog.Alert("tcpServer start:",ipPort)

	listener,err :=net.Listen("tcp",ipPort)
	if err !=nil{
		mylog.Error("net.Listen tcp err:",err.Error())
		zlib.PanicPrint("tcp net.Listen tcp err:"+err.Error())
	}
	mylog.Info("startTcpServer:")
	tcpServer.listener = listener
	tcpServer.Accept()

}

func   (tcpServer *TcpServer)Shutdown( ){
	mylog.Alert("Shutdown tcpServer ")
	err := tcpServer.listener.Close()
	if err != nil{
		mylog.Error("tcpServer.listener.Close err :",err)
	}
}

func (tcpServer *TcpServer)Accept( ){
	for {
		//循环接入所有客户端得到专线连接
		conn,err := tcpServer.listener.Accept()
		if err == nil{
			mylog.Info("listener.Accept new conn:")
		}else{
			mylog.Error("listener.Accept err :",err.Error())
			if strings.Contains(err.Error(), "use of closed network connection") {
				mylog.Warning("TcpAccept end.")
				break
			}else{

			}
			continue
		}
		tcpConn := NewTcpConn(conn)
		//myTcpServer.pool = append(myTcpServer.pool,tcpConn)
		go tcpConn.start()
	}
}
//====================================================================
type TcpConn struct {
	conn net.Conn
	MsgQueue [][]byte
	callbackCloseHandle func(code int, text string)error
	BufferContent string
}

func NewTcpConn(conn net.Conn)*TcpConn{
	mylog.Info("TcpConnNew")
	tcpConn := new (TcpConn)
	tcpConn.conn = conn
	tcpConn.callbackCloseHandle = nil
	return tcpConn
}

func  (tcpConn *TcpConn)start(){
	mylog.Info("TcpConn.start")
	ctx := context.TODO()
	go tcpConn.readLoop(ctx)
	time.Sleep(time.Millisecond * 500)
	myProtocolManager.tcpHandler(tcpConn)
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

func  (tcpConn *TcpConn)readLoop(ctx context.Context ){
	defer func(ctx context.Context ) {
		if err := recover(); err != nil {
			myNetWay.RecoverGoRoutine(tcpConn.readLoop,ctx,err)
		}
	}(ctx)

	mylog.Info("new readLoop:")
	//创建消息缓冲区
	buffer := make([]byte, 1024)
	isBreak := 0
	loopReadCnt := 0
	for {

		if isBreak == 1{
			break
		}
		loopReadCnt++
		//读取客户端发来的消息放入缓冲区
		n,err := tcpConn.conn.Read(buffer)
		if loopReadCnt == 1000{
			if tcpConn.BufferContent != ""{
				mylog.Error("loopReadCnt == 1000 clear")
				tcpConn.BufferContent = ""
			}
			loopReadCnt = 0
		}
		if err != nil{
			mylog.Error("conn.read buffer:",err.Error())
			if err == io.EOF{
				tcpConn.realClose(2)
				isBreak = 1
			}
			continue
		}
		if n == 0{
			continue
		}
		//转化为字符串输出
		clientMsg := buffer[0:n]

		//tcpConn.processMsgBuff(clientMsg)

		mylog.Info("read msg :",n,string(clientMsg))
		//fmt.Printf("收到消息",conn.RemoteAddr(),clientMsg)
		tcpConn.MsgQueue = append(tcpConn.MsgQueue,clientMsg)
		tcpConn.conn.Write([]byte("im server"))
	}
}
//防止粘包，每条消息加上分隔符
func  (tcpConn *TcpConn)processMsgBuff(buffMsg []byte){
	if len( tcpConn.BufferContent) > 10240 {
		mylog.Error("tcpConn.BufferContent > 1024B clear.")
		tcpConn.BufferContent = ""
	}
	for i:=0;i<len(buffMsg);i++{
		if string(buffMsg[i]) == TCP_MSG_SEPARATOR{
			oneMsg := tcpConn.BufferContent
			tcpConn.MsgQueue = append(tcpConn.MsgQueue,[]byte(oneMsg))
			tcpConn.BufferContent = ""
		}else{
			tcpConn.BufferContent +=  string(buffMsg[i])
		}
	}
	//if len(buffMsg) <= len(TCP_MSG_SEPARATOR) {
	//	tcpConn.BufferContent = tcpConn.BufferContent + string(buffMsg)
	//	return
	//}


	//if tcpConn.BufferContent != ""{//之前还有buff未接收全的数据，将：第一个元素得取出来，跟上一个包合并一下
	//	tcpConn.BufferContent = tcpConn.BufferContent + string(buffMsg[0])
	//	if len(msgArr) == 1{
	//		return
	//	}
	//	msgArr = msgArr[1:]
	//}
	//msgArr := strings.Split(string(buffMsg),TCP_MSG_SEPARATOR)
	////取出最后一个元素
	//lastElement := msgArr[len(msgArr) - 1]
	//if string(lastElement) == ""{
	//	for i:=0;i<len(msgArr)-1;i++{
	//		tcpConn.MsgQueue = append(tcpConn.MsgQueue,[]byte(msgArr[i]))
	//	}
	//}else{
	//	if len(msgArr) == 1{
	//		tcpConn.BufferContent = tcpConn.BufferContent + string(buffMsg)
	//		return
	//	}else{
	//
	//	}
	//}
	//
	//tcpConn.BufferContent = tcpConn.BufferContent + string(lastElement)
	//lastLocation := len(msgArr) - 1 - 1
	//msgArr = msgArr[0:lastLocation]
	//
	//for i:=0;i<len(msgArr)-1;i++{
	//	tcpConn.MsgQueue = append(tcpConn.MsgQueue,[]byte(msgArr[i]))
	//}



	//lastLocation := len(buffMsg)-len(TCP_MSG_SEPARATOR)
	//lastStr := buffMsg[lastLocation: ]
	//if string(lastStr) != TCP_MSG_SEPARATOR{
	//
	//}
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