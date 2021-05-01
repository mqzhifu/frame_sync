package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/golang/protobuf/proto"
	"strconv"
	"time"
	"zlib"
)


type WsConnManager struct {
	Pool map[int32]*WsConn	//ws 连接池
}

type WsConn struct {
	AddTime		int32
	UpTime 		int32
	PlayerId	int32
	Status  	int
	Conn 		*websocket.Conn
	CloseChan 	chan int
	RTT 		int64
	MsgInChan		chan Message
	//outChan 	chan []byte
}

func NewWsConnManager()*WsConnManager{
	wsConnManager :=  new(WsConnManager)
	//全局变量
	wsConnManager.Pool = make(map[int32]*WsConn)

	return wsConnManager
}

func (wsConnManager *WsConnManager)CreateOneWsConn(conn *websocket.Conn)(myWsConn *WsConn,err error ){
	mylog.Info("Create one WsConn  client struct")
	if int32(len(wsConnManager.Pool))   > mynetWay.Option.MaxClientConnNum{
		mylog.Error("more MaxClientConnNum")
		return myWsConn,errors.New("more MaxClientConnNum")
	}
	now := int32(zlib.GetNowTimeSecondToInt())

	myWsConn = new (WsConn)
	myWsConn.Conn 		= conn
	myWsConn.PlayerId 	= 0
	myWsConn.AddTime 	= now
	myWsConn.UpTime 	= now
	myWsConn.Status  	= CONN_STATUS_WAITING
	myWsConn.MsgInChan  = make(chan Message,5000)
	//myWsConn.inChan =  make(chan []byte, 1000)
	//myWsConn.outChan=  make(chan []byte,1000)
	//ConnPollNoAuth[ConnPollNoAuthLen] = myWsConn

	mylog.Info("reg wsConn callback CloseHandler")
	conn.SetCloseHandler(myWsConn.CloseHandler)

	return myWsConn,nil
}
func (wsConnManager *WsConnManager)getConnPoolById(id int32)(*WsConn,bool){
	wsConn,ok := wsConnManager.Pool[id]
	return wsConn,ok
}
func  (wsConnManager *WsConnManager)addConnPool( NewWsConn *WsConn)error{
	v ,exist := wsConnManager.getConnPoolById(NewWsConn.PlayerId)
	if exist{
		msg := strconv.Itoa(int(NewWsConn.PlayerId)) + " player has joined conn pool ,addTime : "+strconv.Itoa(int(v.AddTime)) + " , u can , kickOff old fd.?"
		mylog.Warning(msg)
		err := errors.New(msg)
		return err
	}
	mylog.Info("addConnPool : ",NewWsConn.PlayerId)
	wsConnManager.Pool[NewWsConn.PlayerId] = NewWsConn
	return nil
}

func  (wsConnManager *WsConnManager)delConnPool(uid int32  ){
	mylog.Warning("delConnPool uid :",uid)
	delete(wsConnManager.Pool,uid)
}


func CompressContent(contentStruct interface{})(content []byte  ,err error){
	if mynetWay.Option.ContentType == CONTENT_TYPE_JSON{
		content,err = json.Marshal(contentStruct)
	}else if  mynetWay.Option.ContentType == CONTENT_TYPE_PROTOBUF{
		contentStruct := contentStruct.(proto.Message)
		content, err = proto.Marshal(contentStruct)
	}else{
		err = errors.New(" switch err")
	}
	if err != nil{
		mylog.Error("Compress err :",err.Error())
	}
	return content,err
}

//发送一条消息，此方法是给:UID还未注册到池里的情况 ，主要是首次登陆验证出错 的时候
func   (wsConn *WsConn)SendMsg(contentStruct interface{}){
	contentByte ,_ := CompressContent(contentStruct)
	wsConn.Write(contentByte)
}

func   (wsConn *WsConn)Write(content []byte){
	wsConn.Conn.WriteMessage(websocket.TextMessage,[]byte(content))
	//go NewWsConn.outChan
}
func   (wsConn *WsConn)UpLastTime(){
	wsConn.UpTime = int32( zlib.GetNowTimeSecondToInt() )
}

func   (wsConn *WsConn)ReadBinary()(content []byte,err error){
	messageType , dataByte  , err  := wsConn.Conn.ReadMessage()
	if err != nil{
		mynetWay.Option.Mylog.Error("wsConn.Conn.ReadMessage err: ",err.Error())
		return content,err
	}
	mylog.Debug("WsConn.ReadMessage Binary messageType:",messageType , " len :",len(dataByte) , " data:" ,string(dataByte))
	//content = string(dataByte)
	return dataByte,nil
}

func   (wsConn *WsConn)Read()(content string,err error){
	// 设置消息的最大长度 - 暂无
	//wsConn.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(mynetWay.Option.IOTimeout)))
	messageType , dataByte  , err  := wsConn.Conn.ReadMessage()
	if err != nil{
		mynetWay.Option.Mylog.Error("wsConn.Conn.ReadMessage err: ",err.Error())
		return content,err
	}
	mylog.Debug("WsConn.ReadMessage messageType:",messageType , " len :",len(dataByte) , " data:" ,string(dataByte))
	content = string(dataByte)
	return content,nil
}

func  (wsConn *WsConn)IOLoop(){
	mynetWay.Option.Mylog.Info("IOLoop:")
	mynetWay.Option.Mylog.Info("set wsConn status :",CONN_STATUS_EXECING, " make close chan")
	wsConn.Status = CONN_STATUS_EXECING
	wsConn.CloseChan = make(chan int)
	ctx,cancel := context.WithCancel(mynetWay.Option.Cxt)
	go wsConn.ReadLoop(ctx)
	go wsConn.ProcessMsgLoop(ctx)
	<- wsConn.CloseChan
	mynetWay.Option.Mylog.Warning("IOLoop receive chan quit~~~")
	cancel()
}
func  (wsConn *WsConn)ReadLoop(ctx context.Context){
	for{
		select{
		case <-ctx.Done():
			goto end
		default:
			//从ws 读取 数据
			content,err :=  wsConn.Read()
			if err != nil{
				IsUnexpectedCloseError := websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure)
				mylog.Warning("WsConnReadLoop WsConnRead err:",err.Error(),"IsUnexpectedCloseError:",IsUnexpectedCloseError)
				if IsUnexpectedCloseError{
					mynetWay.CloseOneConn(wsConn,CLOSE_SOURCE_CLIENT_WS_FD_GONE)
					goto end
				}else{
					continue
				}
			}

			if content == ""{
				continue
			}

			wsConn.UpLastTime()
			msg,err  := mynetWay.parserContent(content)
			if err !=nil{
				mylog.Warning("parserContent err :",err.Error())
				continue
			}
			wsConn.MsgInChan <- msg
		}
	}
end :
	mynetWay.Option.Mylog.Warning("WsConnReadLoop receive signal: done.")
}

func  (wsConn *WsConn)ProcessMsgLoop(ctx context.Context){
	for{
		select{
			case <-ctx.Done():
				goto end
			case msg := <-wsConn.MsgInChan:
				mylog.Info("ProcessMsgLoop receive msg",msg.Action)
				mynetWay.Router(msg,wsConn)
			default:
		}
	}
end :
	mynetWay.Option.Mylog.Warning("ProcessMsgLoop receive signal: done.")
}


func (wsConnManager *WsConnManager)checkConnPoolTimeout(ctx context.Context){
	mylog.Info("checkConnPoolTimeout start:")
	for{
		select {
		case   <-ctx.Done():
			goto end
		default:
			for _,v := range wsConnManager.Pool{
				now := int32 (zlib.GetNowTimeSecondToInt())
				x := v.UpTime + mynetWay.Option.ConnTimeout
				if now  > x {
					mynetWay.CloseOneConn(v,CLOSE_SOURCE_TIMEOUT)
				}
			}
			time.Sleep(time.Second * 1)
			//mySleepSecond(1,"checkConnPoolTimeout")
		}
	}
end:
	mylog.Warning("checkConnPoolTimeout close")
}