package main

import (
	"errors"
	"github.com/gorilla/websocket"
	"zlib"
)

type WsConn struct {
	AddTime		int
	UpTime 		int
	PlayerId	int
	Status  	int
	Conn 		*websocket.Conn
	CloseChan 	chan int
	RTT 		int64
	//inChan		chan []byte
	//outChan 	chan []byte
}

func (netWay *NetWay)CreateOneWsConn(conn *websocket.Conn)(myWsConn *WsConn,err error ){
	netWay.Option.Mylog.Info("Create one WsConn  client struct")
	if len(ConnPool)   > netWay.Option.MaxClientConnNum{
		netWay.Option.Mylog.Error("more MaxClientConnNum")
		return myWsConn,errors.New("more MaxClientConnNum")
	}
	now := zlib.GetNowTimeSecondToInt()

	myWsConn = new (WsConn)
	myWsConn.Conn 		= conn
	myWsConn.PlayerId 	= 0
	myWsConn.AddTime 	= now
	myWsConn.UpTime 	= now
	myWsConn.Status  	= CONN_STATUS_WAITING
	//myWsConn.inChan =  make(chan []byte, 1000)
	//myWsConn.outChan=  make(chan []byte,1000)
	//ConnPollNoAuth[ConnPollNoAuthLen] = myWsConn

	netWay.Option.Mylog.Info("reg wsConn callback CloseHandler")
	conn.SetCloseHandler(myWsConn.CloseHandler)

	return myWsConn,nil
}

func   (wsConn *WsConn)Write(content string){
	wsConn.Conn.WriteMessage(websocket.TextMessage,[]byte(content))
	//go NewWsConn.outChan

	//send_msg := "[" + string(ReadMsgData[:n]) + "]"
	//m, err := ws.Write([]byte(send_msg))
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Printf("Send: %s\n", ReadMsgData[:m])
	//wsConn.WsConn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	//wsConn.WsConn.Write([]byte(content))
}
func   (wsConn *WsConn)Read()(msg Message,empty bool,err error){
	// 设置消息的最大长度 - 暂无

	//wsConn.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(mynetWay.Option.IOTimeout)))
	content := ""
	//for {
	messageType , dataByte  , err  := wsConn.Conn.ReadMessage()
	if err != nil{
		mynetWay.Option.Mylog.Error("wsConn.Conn.ReadMessage err: ",err.Error())
		return msg,true,err

		//	log.Println("消息读取出现错误", err.Error())
		//	wsConn.close()
		//	return
		//break
	}
	mylog.Debug("WsConn.ReadMessage messageType:",messageType , " dataByte:" ,string(dataByte))
	//if len(dataByte) == 0{
	//	break
	//}
	content += string(dataByte)

	//	req := &wsMessage{
	//		msgType,
	//		data,
	//	}
	//	// 放入请求队列,消息入栈
	//	select {
	//	case wsConn.inChan <- req:
	//	case <-wsConn.closeChan:
	//		return
	//	}
	//}
	//myWs.Option.Mylog.Info("WsConnRead:",content)
	if content != ""{
		msg := mynetWay.parserContent(content)
		 wsConn.UpTime = zlib.GetNowTimeSecondToInt()
		return msg,false,nil
	}else{
		return msg,true,nil
	}
}
