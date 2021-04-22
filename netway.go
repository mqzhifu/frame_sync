package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"zlib"
)

type NetWay struct {
	Option NetWayOption
	CloseChan	chan int
	httpServer *http.Server
	MyCtx 		context.Context
	MyCtxCancel context.CancelFunc
	Players *Players
	ProtocolActions *ProtocolActions
}

type NetWayOption struct {
	Host 				string
	Port 				string
	Mylog 				*zlib.Log
	ContentType 		int
	LoginAuthType		string
	LoginAuthSecretKey	string
	MaxClientConnNum	int
	MsgContentMax		int		//一条消息内容最大值
	IOTimeout			int64
	Cxt 				context.Context	//调用方的CTX
	ConnTimeout 		int		//检测FD最后更新时间
	WsUri				string
	Protocol 			int
	MainChan			chan int
	MapSize				int
	RoomPeople			int
	OffLineWaitTime		int	//lockStep 玩家掉线后，其它玩家等待最长时间
}

type Message struct {
	Action  string	`json:"action"`
	Content	string	`json:"content"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有的CORS 跨域请求，正式环境可以关闭
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


var ConnPool 	map[int]*WsConn	//ws 连接池
//下面是快捷全局变量
var mynetWay	*NetWay
var mySync 		*Sync
var myMatch		*Match


func NewNetWay(option NetWayOption)*NetWay{
	option.Mylog.Info("NewNetWay")
	zlib.PrintStruct(option," : ")

	matchOption := MatchOption{
		RoomPeople: option.RoomPeople,
	}
	myMatch = NewMatch(matchOption)
	mySync = NewSync()
	//全局变量
	ConnPool = make(map[int]*WsConn)

	netWay := new(NetWay)
	netWay.Option = option
	netWay.Players = PlayersNew()
	netWay.ProtocolActions = ProtocolActionsNew()

	mynetWay = netWay
	return netWay
}

func (netWay *NetWay)Startup(){
	//从外层调用的CTX上，派生netway自己的根ctx
	startupCtx ,cancel := context.WithCancel(netWay.Option.Cxt)
	netWay.MyCtx = startupCtx
	netWay.MyCtxCancel = cancel

	uri := netWay.Option.WsUri
	netWay.Option.Mylog.Info("ws Startup : ",uri,netWay.Option.Host+":"+netWay.Option.Port)

	dns := netWay.Option.Host + ":" + netWay.Option.Port
	var addr = flag.String("server addr", dns, "server address")
	httpServer := & http.Server{
		Addr:*addr,
	}
	//监听WS请求
	http.HandleFunc(uri,netWay.wsHandler)
	//监听普通HTTP请求
	http.HandleFunc("/www/",wwwHandler)

	netWay.httpServer = httpServer
	netWay.CloseChan = make(chan int)
	go func() {
		<- netWay.CloseChan
		netWay.Quit()

	}()
	go netWay.DemonSignal()//监听信号
	//开启匹配服务
	go myMatch.matchingPlayerCreateRoom  (startupCtx)
	//监听超时的WS连接
	go netWay.checkConnPoolTimeout(startupCtx)

	err := httpServer.ListenAndServe()
	if err != nil {
		netWay.Option.Mylog.Error("ListenAndServe:", err)
	}
}
//一个客户端连接请求进入
func(netWay *NetWay)wsHandler( resp http.ResponseWriter, req *http.Request) {
	netWay.Option.Mylog.Info("wsHandler: have a new client http request")
	//http 升级 ws
	wsConnFD, err := upgrader.Upgrade(resp, req, nil)
	netWay.Option.Mylog.Info("Upgrade this http req to websocket")
	if err != nil {
		netWay.Option.Mylog.Error("Upgrade websocket failed", err.Error())
		return
	}
	//创建一个连接元素，将WS FD 保存到该容器中
	NewWsConn ,err := netWay.CreateOneConnContainer(wsConnFD)
	if err !=nil{
		netWay.Option.Mylog.Error(err.Error())
		NewWsConn.Write(err.Error())
		netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_CREATE)
		return
	}
	msg,empty,err := NewWsConn.WsConnRead()
	if empty{
		netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_FD_READ_EMPTY)
		return
	}
	if err != nil {
		netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_FD_READ_ERR)
		return
	}

	jwtDataInterface,err := mynetWay.Router(msg,NewWsConn)
	jwtData := jwtDataInterface.(zlib.JwtData)
	netWay.Option.Mylog.Debug("login rs :",jwtData,err)
	if err != nil{
		netWay.Option.Mylog.Error(err.Error())
		NewWsConn.Write(err.Error())
		netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_AUTH_FAILED)
		return
	}
	netWay.Option.Mylog.Info("login success~~~")

	NewWsConn.PlayerId = jwtData.Payload.Uid
	//将新的连接加入到连接池中，并且与玩家ID绑定
	err = netWay.addConnPool( NewWsConn)
	if err != nil{
		loginRes := ResponseLoginRes{Code: 500,Content: err.Error() }
		loginResJsonStr,_ := json.Marshal(loginRes)
		netWay.SendMsgByUid(jwtData.Payload.Uid,"loginRes",string(loginResJsonStr))
		netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_OVERRIDE)
		return
	}
	//给用户再绑定到 用户状态池,该池与连接池的区分 是：连接一但关闭，该元素即删除~而用户状态得需要保存
	existRoomId := netWay.Players.addPlayerPool(jwtData.Payload.Uid)
	var loginRes ResponseLoginRes
	if existRoomId != ""{
		loginRes = ResponseLoginRes{Code: 201,Content: existRoomId }
	}else{
		loginRes = ResponseLoginRes{Code: 200,Content: "ok" }
	}
	loginResJsonStr,_ := json.Marshal(loginRes)
	netWay.SendMsgByUid(jwtData.Payload.Uid,"loginRes",string(loginResJsonStr))
	//初始化即登陆成功的响应均完成后，开始该连接的 读取协程
	go NewWsConn.IOLoop()
	netWay.pintRTT(jwtData.Payload.Uid)
}

func  (netWay *NetWay)pintRTT(playerId int){
	//ping 一下，测试下RTT
	ss := time.Now().UnixNano() / 1e6
	PingRTT := RequestPingRTT{
		AddTime:ss,
	}
	PingRTTJsonStr ,_:=json.Marshal(PingRTT)
	netWay.SendMsgByUid(playerId,"serverPing",string(PingRTTJsonStr))
}

func  (netWay *NetWay)parserContent(content string)Message{
	actionIdStr := content[0:4]
	actionId,_ := strconv.Atoi(actionIdStr)
	actionName,empty := netWay.ProtocolActions.GetActionName(actionId,"client")
	if empty{
		mylog.Error("parserContent actionId no match",actionId)
	}
	msg := Message{
		Action: actionName.Action,
		Content: content[4:],
	}
	//switch netWay.Option.ContentType {
	//case CONTENT_TYPE_JSON:
	//	err := json.Unmarshal([]byte(content),&msg)
	//	if err != nil{
	//		netWay.Option.Mylog.Error("json.Unmarshal err : ",err.Error())
	//	}
	//default:
	//	mylog.Error("content type err:",netWay.Option.ContentType)
	//}
	return msg
}

func (netWay *NetWay)addConnPool( NewWsConn *WsConn)error{
	v ,exist := ConnPool[NewWsConn.PlayerId]
	if exist{
		msg := strconv.Itoa(NewWsConn.PlayerId) +  " has joined conn pool ,addTime : "+strconv.Itoa(v.AddTime)
		netWay.Option.Mylog.Warning("playerId : " , v.PlayerId)
		err := errors.New(msg)
		return err
		//netWay.CloseOneConn(v,CLOSE_SOURCE_OVERRIDE)
	}
	netWay.Option.Mylog.Info("addConnPool : ",NewWsConn.PlayerId)
	ConnPool[NewWsConn.PlayerId] = NewWsConn
	return nil
}
func  (netWay *NetWay)delConnPool(uid int  ){
	netWay.Option.Mylog.Warning("delConnPool uid :",uid)
	delete(ConnPool,uid)
}


func  (wsConn *WsConn)CloseHandler(code int, text string) error{
	mynetWay.CloseOneConn(wsConn,CLOSE_SOURCE_CLIENT)
	return nil
}

func(netWay *NetWay)SendMsgByUid(uid int,action string , content string){
	//msg :=  Msg{
	//	Action: action,
	//	Content: content,
	//}
	//netWay.Option.Mylog.Info("SendMsgByUid :",uid)
	//jsonContent,_ := json.Marshal(msg)
	actionMapT,empty := netWay.ProtocolActions.GetActionId(action,"server")
	mylog.Info("SendMsgByUid",actionMapT.Id,uid,action,content)
	if empty{
		mylog.Error("GetActionId empty",action)
	}
	content = strconv.Itoa(actionMapT.Id) + content
	wsConn,ok := ConnPool[uid]
	if !ok {
		mylog.Error("wsConn not in pool,maybe del.")
		return
	}
	if wsConn.Status == CONN_STATUS_CLOSE{
		mylog.Error("wsConn status =CONN_STATUS_CLOSE.")
		return
	}
	wsConn.Conn.WriteMessage(websocket.TextMessage,[]byte(content))
}

func(netWay *NetWay)login(requestLogin RequestLogin,wsConn *WsConn)(JwtData zlib.JwtData,err error){
	token := ""
	if netWay.Option.LoginAuthType == "jwt"{
		token = requestLogin.Token
		jwtData,err := zlib.ParseJwtToken(netWay.Option.LoginAuthSecretKey,token)
		return jwtData,err
	}else{
		netWay.Option.Mylog.Error("LoginAuthType err")
	}

	return JwtData,err
}
func  (wsConn *WsConn)IOLoop(){
	mynetWay.Option.Mylog.Info("IOLoop:")
	mynetWay.Option.Mylog.Info("set wsConn status :",CONN_STATUS_EXECING, " make close chan")
	wsConn.Status = CONN_STATUS_EXECING
	wsConn.CloseChan = make(chan int)
	ctx,cancel := context.WithCancel(mynetWay.Option.Cxt)
	go wsConn.WsConnReadLoop(ctx)
	<- wsConn.CloseChan
	mynetWay.Option.Mylog.Warning("IOLoop receive chan quit~~~")
	cancel()
}
func  (wsConn *WsConn)WsConnReadLoop(ctx context.Context){
	i := 0
	for{
		select{
		case <-ctx.Done():
			goto end
		default:
			msg,empty,err :=  wsConn.WsConnRead()
			if empty{
				mylog.Warning("WsConnReadLoop WsConnRead empty~")
				time.Sleep(time.Second * 1)
				continue
			}
			if err != nil{
				mylog.Warning("WsConnReadLoop WsConnRead err:",err.Error())
				time.Sleep(time.Second * 1)
				continue
			}
			mynetWay.Router(msg,wsConn)
			i++
			//if i > 3 {
			//	break
			//}
		}
	}
	end :
		mynetWay.Option.Mylog.Warning("WsConnReadLoop receive signal: done.")
}

func (netWay *NetWay)CloseOneConn(wsConn *WsConn,source int){
	netWay.Option.Mylog.Info("wsConn close ,source : ",source)
	if wsConn.Status == CONN_STATUS_EXECING{
		wsConn.Status = CONN_STATUS_CLOSE
		wsConn.CloseChan <- 1
	}else{
		netWay.Option.Mylog.Error("wsConn.Status error")
		return
	}
	mySync.close(wsConn)

	err := wsConn.Conn.Close()
	netWay.Option.Mylog.Info("wsConn.Conn.Close err:",err)

	netWay.delConnPool(wsConn.PlayerId)
	myMatch.delOnePlayer(wsConn.PlayerId)
	//mySleepSecond(2,"CloseOneConn")
}

func(netWay *NetWay)playerReady(requestPlayerReady RequestPlayerReady ,wsConn *WsConn) {
	//给报名池添加用户，以快速成一局游戏
	err := myMatch.addOnePlayer(wsConn.PlayerId)
	if err != nil{
		mylog.Error("playerReady",err.Error())
		//loginRes := LoginRes{Code: 501,Content: err.Error() }
		//loginResJsonStr,_ := json.Marshal(loginRes)
		//netWay.SendMsgByUid(jwtData.Payload.Uid,"loginRes",string(loginResJsonStr))
		//netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_OVERRIDE)
		return
	}
}
func(netWay *NetWay) ClientPong(PingRTT RequestPingRTT,wsConn *WsConn){

	RTT := PingRTT.ClientReceiveTime -  PingRTT.AddTime
	wsConn.RTT = RTT
	mylog.Info("client RTT:",RTT," ms")
}

func RouterJsonUnmarshal(content string ,out interface{}){
	err := json.Unmarshal([]byte(content),out)
	if err != nil{
		mylog.Error("RouterJsonUnmarshal err ",err.Error())
	}
}

func(netWay *NetWay) Router(msg Message,wsConn *WsConn)(data interface{},err error){
	switch msg.Action {
		case "login":
			requestLogin := RequestLogin{}
			RouterJsonUnmarshal(msg.Content,&requestLogin)
			//json.Unmarshal([]byte(msg.Content),&requestLogin)
			return  mynetWay.login(requestLogin,wsConn)
		case "clientPong"://
			PingRTT :=RequestPingRTT{}
			RouterJsonUnmarshal(msg.Content,&PingRTT)
			netWay.ClientPong(PingRTT,wsConn)
		case "playerResumeGame"://恢复未结束的游戏
			requestPlayerResumeGame := RequestPlayerResumeGame{}
			RouterJsonUnmarshal(msg.Content,&requestPlayerResumeGame)
			mySync.playerResumeGame(requestPlayerResumeGame,wsConn )
		case "playerStatus"://玩家检测是否有未结束的游戏
			requestPlayerStatus := RequestPlayerStatus{}
			RouterJsonUnmarshal(msg.Content,&requestPlayerStatus)
			netWay.Players.getPlayerStatus(requestPlayerStatus,wsConn)
		case "playerCommandPush"://玩家推送操作指令
			logicFrame := LogicFrame{}
			RouterJsonUnmarshal(msg.Content,&logicFrame)
			mySync.receiveCommand(logicFrame,wsConn)
		case "playerLogicFrameAck":
			logicFrame := LogicFrame{}
			RouterJsonUnmarshal(msg.Content,&logicFrame)
			mySync.playerLogicFrameAck(logicFrame,wsConn)
		case "playerCancelReady"://玩家取消报名等待
			RequestCancelSign := RequestCancelSign{}
			RouterJsonUnmarshal(msg.Content,&RequestCancelSign)
			mySync.cancelSign(RequestCancelSign,wsConn)
		case "gameOver"://游戏结束
			requestGameOver := RequestGameOver{}
			RouterJsonUnmarshal(msg.Content,&requestGameOver)
			mySync.gameOver(requestGameOver,wsConn)
		case "clientHeartbeat"://心跳
			requestClientHeartbeat := RequestClientHeartbeat{}
			RouterJsonUnmarshal(msg.Content,&requestClientHeartbeat)
			netWay.heartbeat(requestClientHeartbeat,wsConn)
		case "playerReady"://玩家进入状态状态
			requestPlayerReady := RequestPlayerReady{}
			RouterJsonUnmarshal(msg.Content,&requestPlayerReady)
			netWay.playerReady(requestPlayerReady,wsConn)
		//case "netClose"://网络异常断开，也可能是主动断开
		//case "clientPreClose"://C端主动断开连接前，提前通知
		//	netWay.CloseOneConn(wsConn,CLOSE_SOURCE_CLIENT_PRE)
		//case "playerGameOver"://玩家的某些操作，触发了该玩家挂了
		//case "playerAddRoom"://玩家进入房间
		//case "gameStart"://所有-玩家均进入准备状态，点击'开始按钮'，触发游戏开始事件
		default:
			mylog.Error("Router err:",msg)
	}
	return data,nil
}
func(netWay *NetWay)heartbeat(requestClientHeartbeat RequestClientHeartbeat,wsConn *WsConn){
	now := zlib.GetNowTimeSecondToInt()
	wsConn.UpTime = now
}

func(netWay *NetWay)checkConnPoolTimeout(ctx context.Context){
	netWay.Option.Mylog.Info("checkConnPoolTimeout start:")
	for{
		select {
		case   <-ctx.Done():
			goto end
		default:
			for _,v := range ConnPool{
				now := zlib.GetNowTimeSecondToInt()
				if now  >  v.UpTime + netWay.Option.ConnTimeout{
					netWay.CloseOneConn(v,CLOSE_SOURCE_TIMEOUT)
				}
			}
			time.Sleep(time.Second * 1)
			//mySleepSecond(1,"checkConnPoolTimeout")
		}
	}
	end:
		netWay.Option.Mylog.Warning("checkConnPoolTimeout close")
}

func (netWay *NetWay)testCreateJwtToken(){
	jwtDataPayload := zlib.JwtDataPayload{
		Uid : 2,
		ATime: zlib.GetNowTimeSecondToInt(),
		AppId: 1,
		Expire: zlib.GetNowTimeSecondToInt() +  24 * 60 * 60,
	}

	jwtToken := zlib.CreateJwtToken(netWay.Option.LoginAuthSecretKey,jwtDataPayload)
	zlib.ExitPrint(jwtToken)
}
func  (netWay *NetWay)Quit(){
	ctx ,_ := context.WithCancel(netWay.Option.Cxt)
	netWay.httpServer.Shutdown(ctx)

	netWay.MyCtxCancel()//关闭 FD超时检测 、玩家匹配成团

	if len(ConnPool) == 0{
		netWay.Option.Mylog.Warning("ConnPool is 0")
		return
	}
	for _,v := range ConnPool{
		netWay.CloseOneConn(v,CLOSE_SOURCE_SIGNAL_QUIT)
	}
	netWay.Option.Mylog.Warning("quit finish")

}
//信号 处理
func (netWay *NetWay)DemonSignal(){
	mylog.Warning("SIGNAL init : ")
	c := make(chan os.Signal)
	//syscall.SIGHUP :ssh 挂断会造成这个信号被捕获，先注释掉吧
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	prefix := "SIGNAL-DEMON :"
	for{
		sign := <- c
		mylog.Warning(prefix,sign)
		switch sign {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			mylog.Warning(prefix+" exit!!!")
			netWay.Quit()
			goto end
		case syscall.SIGUSR1:
			mylog.Warning(prefix+" usr1!!!")
		case syscall.SIGUSR2:
			mylog.Warning(prefix+" usr2!!!")
		default:
			mylog.Warning(prefix+" unknow!!!")
		}
		mySleepSecond(1,prefix)
	}
	end :
		netWay.Option.Mylog.Warning("DemonSignal end")
		netWay.Option.MainChan <- 1
		netWay.Option.Mylog.Warning("DemonSignal end2")
}
//睡眠 - 协程
func   mySleepSecond(second time.Duration , msg string){
	mylog.Info(msg," sleep second ", strconv.Itoa(int(second)))
	time.Sleep(second * time.Second)
}