package netway

import (
	"github.com/gorilla/websocket"
)

//type PrototolFD struct {
//	WSFD
//	TCPFD 		*net.Conn
//}

type PrototolManager struct {
}
//
func PrototolManagerNew()*PrototolManager{
	prototolManager := new (PrototolManager)
	return prototolManager
}
func (prototolManager *PrototolManager)Start(){
	if mynetWay.Option.Protocol == PROTOCOL_WEBSOCKET{
		//开始HTTP 监听 模块
		go mynetWay.startHttpServer()
	}else if mynetWay.Option.Protocol == PROTOCOL_TCP{
		go myTcpServer.Start()
	}
}

func (prototolManager *PrototolManager)Quit( ){
	//ctx, _ := context.WithCancel(mynetWay.Option.Cxt)
	if mynetWay.Option.Protocol == PROTOCOL_WEBSOCKET{
		mynetWay.httpServer.Shutdown(mynetWay.MyCtx)
	}else if mynetWay.Option.Protocol == PROTOCOL_TCP{
		myTcpServer.Shutdown(mynetWay.MyCtx)
	}
}


//=================================
type FDAdapter interface {
	SetCloseHandler(h func(code int, text string) error)
	WriteMessage(messageType int, data []byte) error
	ReadMessage()(messageType int, p []byte, err error)
	Close()error
}
//===========================
type WebsocketConnImp struct {
	FD	*websocket.Conn
}
func WebsocketConnImpNew(FD *websocket.Conn)*WebsocketConnImp{
	websocketConnImp := new (WebsocketConnImp)
	websocketConnImp.FD = FD
	return websocketConnImp
}

func (websocketConnImp *WebsocketConnImp)SetCloseHandler(h func(code int, text string)error){
	websocketConnImp.FD.SetCloseHandler(h)
}

func (websocketConnImp *WebsocketConnImp)WriteMessage(messageType int, data []byte) error{
	return websocketConnImp.FD.WriteMessage(messageType,data)
}

func (websocketConnImp *WebsocketConnImp)Close()error{
	return websocketConnImp.FD.Close()
}

func (websocketConnImp *WebsocketConnImp)ReadMessage()(messageType int, p []byte, err error){
	return websocketConnImp.FD.ReadMessage()
}

//=========================
type TcpConnImp struct {
	FD	*TcpConn
}
func TcpConnImpNew(FD *TcpConn)*TcpConnImp{
	tcpConnImp := new (TcpConnImp)
	tcpConnImp.FD = FD
	return tcpConnImp
}

func (tcpConnImp *TcpConnImp)SetCloseHandler(h func(code int, text string)error){
	tcpConnImp.FD.SetCloseHandler(h)
}

func (tcpConnImp *TcpConnImp)WriteMessage(messageType int, data []byte) error{
	return tcpConnImp.FD.WriteMessage(messageType,data)
}

func (tcpConnImp *TcpConnImp)Close()error{
	return tcpConnImp.FD.Close()
}

func (tcpConnImp *TcpConnImp)ReadMessage()(messageType int, p []byte, err error){
	return tcpConnImp.FD.ReadMessage()
}