package netway

import (
	"context"
	"errors"
	"frame_sync/myproto"
	"github.com/gorilla/websocket"
	"strconv"
	"sync"
	"time"
	"zlib"
)


type ConnManager struct {
	Pool map[int32]*Conn //ws 连接池
	PoolRWLock *sync.RWMutex
	Close chan int
	MaxClientConnNum int32
	ConnTimeout		int32
}

type Conn struct {
	AddTime			int32
	UpTime 			int32
	PlayerId		int32
	Status  		int
	Conn 			FDAdapter //*websocket.Conn
	CloseChan 		chan int
	RTT 			int64
	MsgInChan		chan myproto.Msg
	RTTCancelChan 	chan int
	SessionId 		string
	UdpConn 		bool
}

func NewConnManager(maxClientConnNum int32,connTimeout int32)*ConnManager {
	mylog.Info("NewConnManager instance:")
	connManager :=  new(ConnManager)
	//全局变量
	connManager.Pool = make(map[int32]*Conn)
	connManager.MaxClientConnNum = maxClientConnNum
	connManager.ConnTimeout = connTimeout
	connManager.Close = make(chan int)
	connManager.PoolRWLock = &sync.RWMutex{}
	return connManager
}
//创建一个新的连接结构体
func (connManager *ConnManager)CreateOneConn(connFd FDAdapter)(myConn *Conn,err error ){
	connManager.PoolRWLock.RLock()
	defer connManager.PoolRWLock.RUnlock()

	mylog.Info("Create one Conn  client struct")
	if int32(len(connManager.Pool))   > connManager.MaxClientConnNum{
		mylog.Error("more MaxClientConnNum")
		return myConn,errors.New("more MaxClientConnNum")
	}
	now := int32(zlib.GetNowTimeSecondToInt())

	myConn = new (Conn)
	myConn.Conn 		= connFd	//*websocket.Conn
	myConn.PlayerId 	= 0
	myConn.AddTime 		= now
	myConn.UpTime 		= now
	myConn.Status  		= CONN_STATUS_INIT
	myConn.MsgInChan  	= make(chan myproto.Msg,5000)
	myConn.UdpConn    	= false
	myConn.RTTCancelChan = make(chan int)

	//mylog.Info("reg conn callback CloseHandler")

	return myConn,nil
}

func (connManager *ConnManager)getConnPoolById(id int32)(*Conn,bool){
	connManager.PoolRWLock.RLock()
	defer connManager.PoolRWLock.RUnlock()

	conn,ok := connManager.Pool[id]
	return conn,ok
}
//往POOL里添加一个新的连接
func  (connManager *ConnManager)addConnPool( NewConn *Conn)error{
	oldConn ,exist := connManager.getConnPoolById(NewConn.PlayerId)
	if exist{
		//msg := strconv.Itoa(int(NewConn.PlayerId)) + " player has joined conn pool ,addTime : "+strconv.Itoa(int(v.AddTime)) + " , u can , kickOff old fd.?"
		msg := strconv.Itoa(int(NewConn.PlayerId)) + " kickOff old pid :"+strconv.Itoa(int(oldConn.PlayerId))
		mylog.Warning(msg)
		//err := errors.New(msg)
		responseKickOff := myproto.ResponseKickOff{
			Time: zlib.GetNowMillisecond(),
		}
		myNetWay.SendMsgCompressByConn(oldConn,"kickOff",&responseKickOff)
		//return err
	}
	mylog.Info("addConnPool : ",NewConn.PlayerId)
	connManager.PoolRWLock.Lock()
	defer connManager.PoolRWLock.Unlock()
	connManager.Pool[NewConn.PlayerId] = NewConn
	return nil
}

func  (connManager *ConnManager)delConnPool(uid int32  ){
	mylog.Warning("delConnPool uid :",uid)
	connManager.PoolRWLock.Lock()
	defer connManager.PoolRWLock.Unlock()

	delete(connManager.Pool,uid)
}

func   (conn *Conn)Write(content []byte,messageType int){
	defer func() {
		if err := recover(); err != nil {
			mylog.Error("conn.Conn.WriteMessage failed:",err)
			myNetWay.CloseOneConn(conn,CLOSE_SOURCE_SEND_MESSAGE)
		}
	}()

	myMetrics.fastLog("total.output.num",METRICS_OPT_INC,0)
	myMetrics.fastLog("total.output.size",METRICS_OPT_PLUS,len(content))

	pid := strconv.Itoa(int(conn.PlayerId))
	myMetrics.fastLog("player.fd.num."+pid,METRICS_OPT_INC,0)
	myMetrics.fastLog("player.fd.size."+pid,METRICS_OPT_PLUS,len(content))

	conn.Conn.WriteMessage(messageType,[]byte(content))
}
func   (conn *Conn)UpLastTime(){
	conn.UpTime = int32( zlib.GetNowTimeSecondToInt() )
}

func   (conn *Conn)ReadBinary()(content []byte,err error){
	messageType , dataByte  , err  := conn.Conn.ReadMessage()
	if err != nil{
		mylog.Error("conn.Conn.ReadMessage err: ",err.Error())
		return content,err
	}
	mylog.Debug("conn.ReadMessage Binary messageType:",messageType , " len :",len(dataByte) , " data:" ,string(dataByte))
	//content = string(dataByte)
	return dataByte,nil
}

func   (conn *Conn)Read()(content string,err error){
	// 设置消息的最大长度 - 暂无
	//conn.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(mynetWay.Option.IOTimeout)))
	//messageType , dataByte  , err  := conn.Conn.ReadMessage()
	_ , dataByte  , err  := conn.Conn.ReadMessage()
	if err != nil{
		myMetrics.fastLog("total.input.err.num",METRICS_OPT_INC,0)
		mylog.Error("conn.Conn.ReadMessage err: ",err.Error())
		return content,err
	}
	myMetrics.fastLog("total.input.num",METRICS_OPT_INC,0)
	myMetrics.fastLog("total.input.size",METRICS_OPT_PLUS,len(dataByte))

	pid := strconv.Itoa(int(conn.PlayerId))
	myMetrics.fastLog("player.fd.num."+pid,METRICS_OPT_INC,0)
	myMetrics.fastLog("player.fd.size."+pid,METRICS_OPT_PLUS,len(content))

	//mylog.Debug("conn.ReadMessage messageType:",messageType , " len :",len(dataByte) , " data:" ,string(dataByte))
	content = string(dataByte)
	return content,nil
}

func  (conn *Conn)IOLoop(){
	mylog.Info("IOLoop:")
	mylog.Info("set conn status :", CONN_STATUS_EXECING, " make close chan")
	conn.Status = CONN_STATUS_EXECING
	conn.CloseChan = make(chan int)
	ctx,cancel := context.WithCancel(myNetWay.Option.OutCxt)
	go conn.ReadLoop(ctx)
	go conn.ProcessMsgLoop(ctx)
	<- conn.CloseChan
	mylog.Warning("IOLoop receive chan quit~~~")
	cancel()
}
func  (conn *Conn) RecoverReadLoop(ctx context.Context){
	mylog.Warning("recover ReadLoop:")
	go conn.ReadLoop(ctx)
}
func  (conn *Conn)ReadLoop(ctx context.Context){
	defer func(ctx context.Context) {
		if err := recover(); err != nil {
			mylog.Panic("conn.ReadLoop panic defer :",err)
			conn.RecoverReadLoop(ctx)
		}
	}(ctx)
	for{
		select{
		case <-ctx.Done():
			mylog.Warning("connReadLoop receive signal: ctx.Done.")
			goto end
		default:
			//从ws 读取 数据
			content,err :=  conn.Read()
			if err != nil{
				IsUnexpectedCloseError := websocket.IsUnexpectedCloseError(err)
				mylog.Warning("connReadLoop connRead err:",err,"IsUnexpectedCloseError:",IsUnexpectedCloseError)
				if IsUnexpectedCloseError{
					myNetWay.CloseOneConn(conn, CLOSE_SOURCE_CLIENT_WS_FD_GONE)
					goto end
				}else{
					continue
				}
			}

			if content == ""{
				continue
			}

			conn.UpLastTime()
			msg,err  := myProtocolManager.parserContentProtocol(content)
			if err !=nil{
				mylog.Warning("parserContent err :",err.Error())
				continue
			}
			conn.MsgInChan <- msg
		}
	}
end :
	mylog.Warning("connReadLoop receive signal: done.")
}
func  (conn *Conn) RecoverProcessMsgLoop(ctx context.Context){
	mylog.Warning("recover ReadLoop:")
	go conn.ProcessMsgLoop(ctx)
}
func  (conn *Conn)ProcessMsgLoop(ctx context.Context){
	defer func(ctx context.Context) {
		if err := recover(); err != nil {
			mylog.Panic("conn.ProcessMsgLoop panic defer :",err)
			conn.RecoverProcessMsgLoop(ctx)
		}
	}(ctx)

	for{
		ctxHasDone := 0
		select{
			case <-ctx.Done():
				ctxHasDone = 1
			case msg := <-conn.MsgInChan:
				mylog.Info("ProcessMsgLoop receive msg",msg.Action)
				myNetWay.Router(msg,conn)
		}
		if ctxHasDone == 1{
			goto end
		}
	}
end :
	mylog.Warning("ProcessMsgLoop receive signal: done.")
}
//监听到某个FD被关闭后，回调函数
func  (conn *Conn)CloseHandler(code int, text string) error{
	myNetWay.CloseOneConn(conn, CLOSE_SOURCE_CLIENT)
	return nil
}
func (connManager *ConnManager)Shutdown(){
	mylog.Alert("shutdown connManager")
	connManager.Close <- 1
	if len(connManager.Pool) <= 0{
		return
	}
	pool := connManager.getPoolAll( )
	for _,conn :=range pool{
		myNetWay.CloseOneConn(conn,CLOSE_SOURCE_CONN_SHUTDOWN)
	}
}
func (connManager *ConnManager)getPoolAll()map[int32]*Conn{
	connManager.PoolRWLock.RLock()
	defer connManager.PoolRWLock.RUnlock()

	pool := make(map[int32]*Conn)
	for k,v := range connManager.Pool{
		pool[k] = v
	}
	return pool
}
func (connManager *ConnManager)Start(ctx context.Context){
	defer func(ctx context.Context ) {
		if err := recover(); err != nil {
			myNetWay.RecoverGoRoutine(connManager.Start,ctx,err)
		}
	}(ctx)

	mylog.Alert("checkConnPoolTimeout start:")
	for{
		select {
			case   <-connManager.Close:
				goto end
			default:
				pool := connManager.getPoolAll()
				for _,v := range pool{
					now := int32 (zlib.GetNowTimeSecondToInt())
					x := v.UpTime + connManager.ConnTimeout
					if now  > x {
						myNetWay.CloseOneConn(v, CLOSE_SOURCE_TIMEOUT)
					}
				}
				time.Sleep(time.Second * 1)
			//mySleepSecond(1,"checkConnPoolTimeout")
		}
	}
end:
	mylog.Alert(CTX_DONE_PRE+"checkConnPoolTimeout close")
}

