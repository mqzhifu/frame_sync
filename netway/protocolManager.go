package netway

import (
	"context"
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
)

//type PrototolFD struct {
//	WSFD
//	TCPFD 		*net.Conn
//}

type PrototolManager struct {
	TcpServer *TcpServer
}
//
func PrototolManagerNew()*PrototolManager{
	prototolManager := new (PrototolManager)
	return prototolManager
}
func (prototolManager *PrototolManager)Start(outCtx context.Context){
	//开始HTTP 监听 模块
	go prototolManager.startHttpServer()
	//tcp server
	myTcpServer :=  TcpServerNew(outCtx)
	prototolManager.TcpServer = myTcpServer
	go myTcpServer.Start()
}

//启动HTTP 服务
func (prototolManager *PrototolManager)startHttpServer( ){
	mynetWay.Option.Mylog.Info("ws Startup : ",mynetWay.Option.ListenIp+":"+mynetWay.Option.WsPort,mynetWay.Option.WsUri)

	dns := mynetWay.Option.ListenIp + ":" + mynetWay.Option.WsPort
	var addr = flag.String("server addr", dns, "server address")

	logger := log.New(os.Stdout,"h_s_err",log.Ldate)
	httpServer := & http.Server{
		Addr:*addr,
		ErrorLog: logger,
	}
	//监听WS请求
	http.HandleFunc(mynetWay.Option.WsUri,mynetWay.wsHandler)
	//监听普通HTTP请求
	http.HandleFunc("/www/", wwwHandler)

	mynetWay.httpServer = httpServer
	err := httpServer.ListenAndServe()
	if err != nil {
		mynetWay.Option.Mylog.Error(" ListenAndServe err:", err)
	}
}

func (prototolManager *PrototolManager)Quit( startupCtx context.Context){
	//ctx, _ := context.WithCancel(mynetWay.Option.Cxt)
	//if mynetWay.Option.Protocol == PROTOCOL_WEBSOCKET{
		mynetWay.httpServer.Shutdown(startupCtx)
		mylog.Alert(CTX_DONE_PRE + " httpServer")
	//}else if mynetWay.Option.Protocol == PROTOCOL_TCP{
		prototolManager.TcpServer.Shutdown()
	//}
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