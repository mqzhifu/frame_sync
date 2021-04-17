package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"zlib"
)

const (
	CLOSE_SOURCE_CLIENT = 1			//客户端断开连接
	CLOSE_SOURCE_AUTH_FAILED = 2	//服务端验证失败
	CLOSE_SOURCE_CREATE = 3			//初始化 连接类失败，可能是连接数过大
	CLOSE_SOURCE_CLIENT_PRE = 4		//C端主动，预先关闭
	CLOSE_SOURCE_OVERRIDE = 5		//创建新连接时，发现，该用户还有一个未关闭的连接
	CLOSE_SOURCE_TIMEOUT = 6		//最后更新时间 ，超时
	CLOSE_SOURCE_SIGNAL_QUIT = 7 	//接收到关闭信号
	CLOSE_SOURCE_FD_READ_EMPTY = 8
	CLOSE_SOURCE_FD_READ_ERR = 9

	CONTENT_TYPE_JSON = 1			//内容类型 json
	CONTENT_TYPE_PROTOBUF = 2		//proto_buf

	CONN_STATUS_WAITING = 1
	CONN_STATUS_EXECING = 2		//

	PROTOCOL_SOCKET = 1
	PROTOCOL_WEBSOCKET = 2

	ROOM_STATUS_WAIT = 1
	ROOM_STATUS_WAIT_EXECING = 2
	ROOM_STATUS_WAIT_END = 3

	//一个副本的整体状态
	SYNC_ELEMENT_STATUS_WAIT = 1
	SYNC_ELEMENT_STATUS_EXECING = 2
	SYNC_ELEMENT_STATUS_END = 3
	//一个副本的，一条消息的，同步状态
	PLAYERS_ACK_STATUS_INIT = 1
	PLAYERS_ACK_STATUS_WAIT = 2
	PLAYERS_ACK_STATUS_OK = 3


	PLAYER_STATUS_ONLINE = 1
	PLAYER_STATUS_OFFLINE = 2
)

type NetWay struct {
	Option NetWayOption
	CloseChan	chan int
	httpServer *http.Server
	MyCtx 		context.Context
	MyCtxCancel context.CancelFunc
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

type Msg struct {
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

type Player struct {
	Id 			int
	Nickname	string
	Status 		int
	AddTime		int
	UpTime 		int
	RoomId 		string
}

type LoginRes struct {
	Code int		`json:"code"`
	Content string `json:"content"`
}

type ActionMap struct {
	Id 		int
	Action	string
	Desc 	string
	Demo 	string
}

var PlayerPool map[int]*Player
var ConnPool 	map[int]*WsConn
//下面是快捷全局变量
//var wsLog		*zlib.Log
var mynetWay	*NetWay
var mySync 		*Sync
var myMatch		*Match
var actionMap  	map[string]map[int]ActionMap
func NewNetWay(option NetWayOption)*NetWay{
	option.Mylog.Info("NewNetWay")
	zlib.PrintStruct(option," : ")

	ConnPool = make(map[int]*WsConn)
	netWay := new(NetWay)
	netWay.Option = option

	PlayerPool = make(map[int]*Player)

	matchOption := MatchOption{
		RoomPeople: option.RoomPeople,
	}

	myMatch = NewMatch(matchOption)
	mynetWay = netWay
	//wsLog = option.Mylog
	mySync = NewSync()
	//mynetWay.testCreateJwtToken()
	netWay.initActionMap()
	return netWay
}



func (netWay *NetWay)initActionMap(){
	mylog.Info("initActionMap")
	actionMap = make( 	map[string]map[int]ActionMap )

	actionMap["client"] = loadingFile("clientActionMap.txt")
	actionMap["server"] = loadingFile("serverActionMap.txt")
}


func loadingFile(fileName string)map[int]ActionMap{
	client,err := ReadLine(fileName)
	if err != nil{
		zlib.ExitPrint("initActionMap ReadLine err :",err.Error())
	}
	am := make(map[int]ActionMap)
	for _,v:= range client{
		contentArr := strings.Split(v,"|")
		id := zlib.Atoi(contentArr[1])
		//zlib.ExitPrint(id)
		actionMap := ActionMap{
			Id: id,
			Action: contentArr[2],
			Desc: contentArr[3],
			Demo: contentArr[4],
		}
		am[id] = actionMap
	}
	return am
}

func ReadLine(fileName string) ([]string,error){
	f, err := os.Open(fileName)
	if err != nil {
		return nil,err
	}
	buf := bufio.NewReader(f)
	var result []string
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil {
			if err == io.EOF { //读取结束，会报EOF
				return result,nil
			}
			return nil,err
		}
		result = append(result,line)
	}
	return result,nil
}

func (netWay *NetWay)getActionMap()map[string]map[int]ActionMap{
	return actionMap
}

func (netWay *NetWay)Startup(){
	//从外层调用的CTX上，派生netway自己的根ctx
	startupCtx ,cancel := context.WithCancel(netWay.Option.Cxt)
	netWay.MyCtx = startupCtx
	netWay.MyCtxCancel = cancel

	uri := netWay.Option.WsUri
	netWay.Option.Mylog.Info("ws Startup : ",uri,netWay.Option.Host+":"+netWay.Option.Port)
	//http.HandleFunc(uri, netWay.wsHandler)

	dns := netWay.Option.Host + ":" + netWay.Option.Port
	var addr = flag.String("server addr", dns, "server address")
	httpServer := & http.Server{
		Addr:*addr,
	}
	http.HandleFunc(uri,netWay.wsHandler)
	http.HandleFunc("/www/",wwwHandler)

	netWay.httpServer = httpServer

	netWay.CloseChan = make(chan int)
	go func() {
		<- netWay.CloseChan
		netWay.Quit()

	}()
	go netWay.DemonSignal()

	go myMatch.matchingPlayerCreateRoom  (startupCtx)
	go netWay.checkConnPoolTimeout(startupCtx)


	//myMatch.addOnePlayer(1)
	//myMatch.addOnePlayer(2)
	//myMatch.addOnePlayer(3)

	err := httpServer.ListenAndServe()
	//err := http.ListenAndServe(netWay.Option.Host+":"+netWay.Option.Port, nil)
	if err != nil {
		netWay.Option.Mylog.Error("ListenAndServe:", err)
	}

}
//func (ws *Ws) wsHandler(wsConn *websocket.Conn) {
func(netWay *NetWay)wsHandler( resp http.ResponseWriter, req *http.Request) {
	netWay.Option.Mylog.Info("wsHandler: have a new client http request")
	//http 升级 ws
	wsConnFD, err := upgrader.Upgrade(resp, req, nil)
	netWay.Option.Mylog.Info("Upgrade this http req to websocket")
	if err != nil {
		netWay.Option.Mylog.Error("Upgrade websocket failed", err.Error())
		return
	}
	//ws.Option.Mylog.Debug("new client websocket")
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
	//jwtData,err := netWay.login(msg)
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
	err = netWay.addConnPoll( NewWsConn)
	if err != nil{
		loginRes := LoginRes{Code: 500,Content: err.Error() }
		loginResJsonStr,_ := json.Marshal(loginRes)
		netWay.SendMsgByUid(jwtData.Payload.Uid,"loginRes",string(loginResJsonStr))
		netWay.CloseOneConn(NewWsConn,CLOSE_SOURCE_OVERRIDE)
		return
	}
	//给用户再绑定到 用户状态池,该池与连接池的区分 是：连接一但关闭，该元素即删除~而用户状态得需要保存
	existRoomId := netWay.addPlayerPool(jwtData.Payload.Uid)
	var loginRes LoginRes
	if existRoomId != ""{
		loginRes = LoginRes{Code: 201,Content: existRoomId }
	}else{
		loginRes = LoginRes{Code: 200,Content: "ok" }
	}
	loginResJsonStr,_ := json.Marshal(loginRes)
	netWay.SendMsgByUid(jwtData.Payload.Uid,"loginRes",string(loginResJsonStr))
	//初始化即登陆成功的响应均完成后，开始该连接的 读取协程
	go NewWsConn.IOLoop()
	//ping 一下，测试下RTT
	ss := time.Now().UnixNano() / 1e6
	//ssString := strconv.FormatInt(ss,10)
	PingRTT := PingRTT{
		AddTime:ss,
	}
	PingRTTJsonStr ,_:=json.Marshal(PingRTT)
	netWay.SendMsgByUid(jwtData.Payload.Uid,"serverPing",string(PingRTTJsonStr))
}
//返回值是roomId,其实只是标注，让C端来获取
func  (netWay *NetWay)addPlayerPool(id int)string{
	mylog.Info("addPlayerPool :",id)
	hasPlayer,exist  := PlayerPool[id]
	if exist{
		mylog.Notice("this player id has exist pool")
		if hasPlayer.Status == PLAYER_STATUS_ONLINE{
			mylog.Error("hasPlayer.Status = PLAYER_STATUS_ONLINE")
		}else{
			netWay.upPlayerPool(id,PLAYER_STATUS_ONLINE)
		}
		return hasPlayer.RoomId
	}else{
		player := Player{
			Id: id,
			AddTime: zlib.GetNowTimeSecondToInt(),
			Nickname: "",
			Status: PLAYER_STATUS_ONLINE,
		}
		PlayerPool[id] = &player
	}
	return ""
}

func  (netWay *NetWay)upPlayerPool(id int,status int){

	player := PlayerPool[id]

	mylog.Info("upPlayerPool" , " old : ",player.Status," new:",status)

	player.Status = PLAYER_STATUS_ONLINE
	player.UpTime = zlib.GetNowTimeSecondToInt()

}

func  (netWay *NetWay)GetActionName(id int,category string)(actionMapT ActionMap,empty bool){
	am := actionMap[category]
	for k,v:=range am{
		if k == id {
			return v,false
		}
	}
	return  actionMapT,true
}

func  (netWay *NetWay)GetActionId(action string,category string)(actionMapT ActionMap,empty bool){
	mylog.Info("GetActionId ",action , " ",category)
	am := actionMap[category]
	for _,v:=range am{
		if v.Action == action {
			return v,false
		}
	}
	return  actionMapT,true
}


func  (netWay *NetWay)parserContent(content string)Msg{
	actionIdStr := content[0:4]
	actionId,_ := strconv.Atoi(actionIdStr)
	actionName,empty := netWay.GetActionName(actionId,"client")
	if empty{
		mylog.Error("parserContent actionId no match",actionId)
	}
	msg := Msg{
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

func (netWay *NetWay)addConnPoll( NewWsConn *WsConn)error{
	v ,exist := ConnPool[NewWsConn.PlayerId]
	if exist{
		msg := strconv.Itoa(NewWsConn.PlayerId) +  " has joined conn poll ,addTime : "+strconv.Itoa(v.AddTime)
		netWay.Option.Mylog.Warning("playerId : " , v.PlayerId)
		err := errors.New(msg)
		return err
		//netWay.CloseOneConn(v,CLOSE_SOURCE_OVERRIDE)
	}
	netWay.Option.Mylog.Info("addConnPoll : ",NewWsConn.PlayerId)
	ConnPool[NewWsConn.PlayerId] = NewWsConn
	return nil
}
func  (netWay *NetWay)delConnPoll(uid int  ){
	netWay.Option.Mylog.Warning("delConnPoll uid :",uid)
	//zlib.MyPrint(len(ConnPool))
	delete(ConnPool,uid)
	//zlib.MyPrint(len(ConnPool))
	//zlib.ExitPrint(111)
}
func (netWay *NetWay)CreateOneConnContainer(conn *websocket.Conn)(myWsConn *WsConn,err error ){
	netWay.Option.Mylog.Info("Create one WsConn  client struct")
	if len(ConnPool)   > netWay.Option.MaxClientConnNum{
		netWay.Option.Mylog.Error("more MaxClientConnNum")
		return myWsConn,errors.New("more MaxClientConnNum")
	}
	//ConnPollNoAuthLen := ws.getConnPollNoAuthInc()

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
	actionMapT,empty := netWay.GetActionId(action,"server")
	mylog.Info("SendMsgByUid",actionMapT.Id,uid,action,content)
	if empty{
		mylog.Error("GetActionId empty",action)
	}
	content = strconv.Itoa(actionMapT.Id) + content
	wsConn := ConnPool[uid]
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

	//zlib.MyPrint("start write")
	////n,err := wsConn.WsConn.Write([]byte("string"))
	//err := wsConn.WsConn.WriteMessage(websocket.TextMessage,[]byte("string"))
	//zlib.MyPrint(err)

}

func (netWay *NetWay)CloseOneConn(wsConn *WsConn,source int){
	netWay.Option.Mylog.Info("wsConn close ,source : ",source)
	if wsConn.Status == CONN_STATUS_EXECING{
		wsConn.CloseChan <- 1
	}else{
		netWay.Option.Mylog.Error("wsConn.Status error")
		return
	}
	mySync.close(wsConn)

	err := wsConn.Conn.Close()
	netWay.Option.Mylog.Info("wsConn.Conn.Close err:",err)

	netWay.delConnPoll(wsConn.PlayerId)
	myMatch.delOnePlayer(wsConn.PlayerId)


	//mySleepSecond(2,"CloseOneConn")
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
func   (wsConn *WsConn)WsConnRead()(msg Msg,empty bool,err error){
	// 设置消息的最大长度 - 暂无

	//wsConn.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(mynetWay.Option.IOTimeout)))
	content := ""
	//for {
		messageType , dataByte  , err  := wsConn.Conn.ReadMessage()
		if err != nil{
			mynetWay.Option.Mylog.Error("wsConn.Conn.ReadMessage err: ",err.Error())
			return msg,true,err
			//	websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure)
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
		return msg,false,nil
	}else{
		return msg,true,nil
	}
}



func(netWay *NetWay)getPlayerStatusById(plyaerId int)(player *Player,empty bool){
	playerStatus ,ok:= PlayerPool[plyaerId]
	if !ok{
		return player,true
	}
	return playerStatus,false
}

func(netWay *NetWay)playerReady(wsConn *WsConn) {
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




func(netWay *NetWay) Router(msg Msg,wsConn *WsConn)(data interface{},err error){
	switch msg.Action {
		case "login":
			requestLogin := RequestLogin{}
			json.Unmarshal([]byte(msg.Content),&requestLogin)
			return  mynetWay.login(requestLogin,wsConn)
		case "clientPong"://
			PingRTT :=PingRTT{}
			err := json.Unmarshal([]byte(msg.Content),&PingRTT)
			if err != nil{
				mynetWay.Option.Mylog.Error("pong Unmarshal ",err.Error())
				break
			}
			RTT := PingRTT.ClientReceiveTime -  PingRTT.AddTime
			wsConn.RTT = RTT
			mylog.Info("client RTT:",RTT," ms")
		case "playerResumeGame"://恢复未结束的游戏
			mySync.playerResumeGame(wsConn )
		case "playerStatus"://玩家检测是否有未结束的游戏
			player,_ := netWay.getPlayerStatusById(wsConn.PlayerId)
			playerStr,_ := json.Marshal(player)
			netWay.SendMsgByUid(wsConn.PlayerId,"pushPlayerStatus",string(playerStr))
		case "playerCommandPush"://玩家推送操作指令
			mySync.receiveCommand(msg.Content,wsConn)
		case "playerLogicFrameAck":
			mySync.playerLogicFrameAck(msg.Content,wsConn)
		case "playerCancelReady"://玩家取消报名等待
			mySync.cancelSign(wsConn)
		case "gameOver"://游戏结束
			mySync.gameOver(msg.Content,wsConn)
		case "clientHeartbeat":
			netWay.heartbeat(msg,wsConn)
	case "playerReady"://玩家进入状态状态
		netWay.playerReady(wsConn)
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
func(netWay *NetWay)heartbeat(msg Msg,wsConn *WsConn){
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

//func (ws *Ws)ConnPollNoAuthMoveConnPoll(connNum int,jwtData zlib.JwtData){
//	ConnPoll[jwtData.Payload.Uid] = ConnPollNoAuth[connNum]
//	delete(ConnPollNoAuth,connNum)
//}

//func aaaa(network, address string, c syscall.RawConn)error{
//	zlib.MyPrint("aaaa",network,address,c)
//	return nil
//}
////Handshake func(*Config, *http.Request) error
//func bbbb( config *websocket.Config ,r *http.Request)error{
//	zlib.MyPrint("bbbb",config,r)
//	return nil
//}