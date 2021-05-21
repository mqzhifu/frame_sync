package netway

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"frame_sync/myproto"
	"frame_sync/myprotocol"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
	"time"
	"unicode"
	"zlib"
)

type NetWay struct {
	Option          NetWayOption
	mySync 			*Sync
	CloseChan       chan int32
	MatchSuccessChan       chan *Room
	httpServer      *http.Server
	MyCtx           context.Context
	MyCtxCancel     context.CancelFunc
	Players         *Players
	ProtocolActions *myprotocol.ProtocolActions
}

type NetWayOption struct {
	Host 				string		`json:"host"`
	Port 				string		`json:"port"`
	Mylog 				*zlib.Log	`json:"mylog"`
	ContentType 		int32		`json:"contentType"`	//json protobuf
	LoginAuthType		string		`json:"loginAuthType"`	//jwt
	LoginAuthSecretKey	string								//密钥
	MaxClientConnNum	int32		`json:"maxClientConnMum"`//客户端最大连接数
	MsgContentMax		int32								//一条消息内容最大值
	IOTimeout			int64								//read write sock fd 超时时间
	Cxt 				context.Context						//调用方的CTX，用于所有协程的退出操作
	MainChan			chan int32	`json:"-"`
	ConnTimeout 		int32								//检测FD最后更新时间
	WsUri				string		`json:"wsUri"`			//接HOST的后面的URL地址
	Protocol 			int32		`json:"myprotocol"`		//协议  ，ws sockdt udp
	MapSize				int32		`json:"mapSize"`		//地址大小，给前端初始化使用
	RoomPeople			int32		`json:"roomPeople"`		//一局游戏包含几个玩家
	RoomTimeout 		int32 		`json:"roomTimeout"`	//一个房间超时时间
	OffLineWaitTime		int32		`json:"offLineWaitTime"`//lockStep 玩家掉线后，其它玩家等待最长时间
	LockMode  			int32 		`json:"lockMode"`		//锁模式，乐观|悲观
	FPS 				int32 		`json:"fps"`			//frame pre second
	RoomReadyTimeout	int32		`json:"roomReadyTimeout"`//一个房间的，玩家的准备，超时时间
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

//下面是全局变量，主要是快捷方便使用，没实际意义
var mynetWay	*NetWay
//var mySync 		*Sync
var myMatch		*Match
var wsConnManager *WsConnManager
var myMetrics *Metrics
var mylog *zlib.Log
func NewNetWay(option NetWayOption)*NetWay {
	option.Mylog.Info("New NetWay instance :")
	zlib.PrintStruct(option," : ")

	mylog = option.Mylog

	matchOption := MatchOption{
		RoomPeople: option.RoomPeople,
	}
	myMatch = NewMatch(matchOption)
	netWay := new(NetWay)

	syncOptions := Options{
		Log: mylog,
		FPS: option.FPS,
		MapSize: option.MapSize,
		LockMode : option.LockMode,
	}
	netWay.mySync = NewSync(syncOptions)
	wsConnManager = NewWsConnManager()


	netWay.Option = option
	netWay.Players = PlayersNew()
	netWay.ProtocolActions = myprotocol.ProtocolActionsNew()

	myMetrics = MetricsNew()

	mynetWay = netWay
	return netWay
}
//启动 - 入口
func (netWay *NetWay)Startup(){
	s_time := zlib.GetNowMillisecond()
	myMetrics.input <- MetricsChanMsg{Key: "starupTime",Opt: 1,Value: int(s_time)}

	//从外层调用的CTX上，派生netway自己的根ctx
	startupCtx ,cancel := context.WithCancel(netWay.Option.Cxt)
	netWay.MyCtx = startupCtx
	netWay.MyCtxCancel = cancel

	netWay.MatchSuccessChan = make(chan *Room)
	//开启匹配服务
	go myMatch.matchingPlayerCreateRoom  (startupCtx,netWay.MatchSuccessChan)
	//监听超时的WS连接
	go wsConnManager.checkConnPoolTimeout(startupCtx)
	//接收<匹配成功>的房间信息，并分发
	go netWay.recviceMatchSuccess(startupCtx)
	//清理，房间到期后，未回收的情况
	//go mySync.checkRoomTimeoutLoop(startupCtx)
	go myMetrics.start(startupCtx)
	//myMetrics.input <- MetricsChanMsg{Key: "startTime",Opt: 1,Value: int( zlib.GetNowMillisecond())}

	netWay.startHttpServer()
}
//启动HTTP 服务
func (netWay *NetWay)startHttpServer( ){
	netWay.Option.Mylog.Info("ws Startup : ",netWay.Option.WsUri,netWay.Option.Host+":"+netWay.Option.Port)

	dns := netWay.Option.Host + ":" + netWay.Option.Port
	var addr = flag.String("server addr", dns, "server address")

	httpServer := & http.Server{
		Addr:*addr,
	}
	//监听WS请求
	http.HandleFunc(netWay.Option.WsUri,netWay.wsHandler)
	//监听普通HTTP请求
	http.HandleFunc("/www/", wwwHandler)

	netWay.httpServer = httpServer
	netWay.CloseChan = make(chan int32)
	go func() {
		<- netWay.CloseChan
		netWay.Quit()
	}()

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
		netWay.Option.Mylog.Error("Upgrade websocket failed: ", err.Error())
		return
	}
	//创建一个连接元素，将WS FD 保存到该容器中
	NewWsConn ,err := wsConnManager.CreateOneWsConn(wsConnFD)
	if err !=nil{
		netWay.Option.Mylog.Error(err.Error())
		NewWsConn.Write([]byte(err.Error()))
		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_CREATE)
		return
	}
	//登陆验证
	jwtData,err := mynetWay.loginPre(NewWsConn)
	if err != nil{
		return
	}
	var loginRes myproto.ResponseLoginRes
	//登陆验证通过，开始添加各种状态及初始化
	NewWsConn.PlayerId = jwtData.Payload.Uid
	//将新的连接加入到连接池中，并且与玩家ID绑定
	err = wsConnManager.addConnPool( NewWsConn)
	if err != nil{
		loginRes = myproto.ResponseLoginRes{
			Code: 500,
			ErrMsg: err.Error(),
		}
		netWay.SendMsgCompressByUid(jwtData.Payload.Uid,"loginRes",loginRes)
		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_OVERRIDE)
		return
	}
	//给用户再绑定到 用户状态池,该池与连接池的区分 是：连接一但关闭，该元素即删除~而用户状态得需要保存
	playerConnInfo ,_ :=netWay.Players.addPlayerPool(jwtData.Payload.Uid)
	loginRes = myproto.ResponseLoginRes{
		Code: 200,
		ErrMsg: "",
		Player: &playerConnInfo,
	}
	netWay.SendMsgCompressByUid(jwtData.Payload.Uid,"loginRes",&loginRes)
	myMetrics.input <- MetricsChanMsg{Key: "total.FDNum",Opt: 2}
	//初始化即登陆成功的响应均完成后，开始该连接的 读取协程
	go NewWsConn.IOLoop()

	//netWay.serverPingRtt(time.Duration(rttMinTimeSecond),NewWsConn,1)
	mylog.Info("wsHandler end ,player login success!!!")
	myMetrics.input <- MetricsChanMsg{Key: "historyFDCreateOK",Opt: 2}
}
func  (netWay *NetWay)loginPre(NewWsConn *WsConn)(jwt zlib.JwtData,err error){
	//这里有个问题，连接成功后，C端立刻就得发消息，不然就异常~bug
	var loginRes myproto.ResponseLoginRes

	content,err := NewWsConn.Read()
	if err != nil{
		loginRes = myproto.ResponseLoginRes{
			Code : 500,
			ErrMsg:err.Error(),
		}
		NewWsConn.SendMsg(loginRes)
		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_FD_READ_EMPTY)
		return
	}
	msg,err := netWay.parserContent(content)
	if err != nil{
		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_FD_PARSE_CONTENT)
		return
	}
	if msg.Action != "login"{
		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_FIRST_NO_LOGIN)
		mylog.Error("first msg must login api!!!")
		return
	}
	//开始：登陆/验证 过程
	jwtDataInterface,err := netWay.Router(msg,NewWsConn)
	jwt = jwtDataInterface.(zlib.JwtData)
	netWay.Option.Mylog.Debug("login rs :",jwt,err)
	if err != nil{
		netWay.Option.Mylog.Error(err.Error())
		loginRes = myproto.ResponseLoginRes{
			Code : 500,
			ErrMsg:err.Error(),
		}
		NewWsConn.SendMsg(loginRes)

		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_AUTH_FAILED)
		return jwt ,err
	}
	netWay.Option.Mylog.Info("login jwt auth ok~~")
	return jwt,nil
}

func(netWay *NetWay)login(requestLogin myproto.RequestLogin,wsConn *WsConn)(JwtData zlib.JwtData,err error){
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

func(netWay *NetWay)recviceMatchSuccess(ctx context.Context){
	mylog.Info("recviceMatchSuccess start:")
	for{
		select {
			case newRoom :=  <-netWay.MatchSuccessChan:
				netWay.mySync.AddPoolElement(newRoom)
				netWay.mySync.Start(newRoom.Id)
			case   <-ctx.Done():
				goto end
			default:
				time.Sleep(time.Second * 1)
			//mySleepSecond(1,"checkConnPoolTimeout")
		}
	}
end:
	mylog.Warning("recviceMatchSuccess close")
}



func(netWay *NetWay)heartbeat(requestClientHeartbeat myproto.RequestClientHeartbeat,wsConn *WsConn){
	now := zlib.GetNowTimeSecondToInt()
	wsConn.UpTime = int32(now)
}
//=================================
//发送一条消息给一个玩家FD，同时将消息内容进行压缩
func(netWay *NetWay)SendMsgCompressByUid(uid int32,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByUid uid:",uid , " action:",action)
	contentByte ,_ := CompressContent(contentStruct)
	netWay.SendMsgByUid(uid,action,contentByte)
}
//发送一条消息给一个玩家FD，
func(netWay *NetWay)SendMsgByUid(uid int32,action string , content []byte){
	//zlib.MyPrint(content)
	actionMapT,empty := netWay.ProtocolActions.GetActionId(action,"server")
	mylog.Info("SendMsgByUid",actionMapT.Id,uid,action)
	if empty{
		mylog.Error("GetActionId empty",action)
	}
	//strid := strconv.Itoa(actionMapT.Id)
	//au := StrToUnicode(strconv.Itoa(actionMapT.Id))
	//arr := []byte(strid)
	//bytecontent := []byte(content)
	//zlib.MyPrint(string(bytecontent))
	//nn := []byte(content)
	//for _,v :=range arr{
	//	ss := StrToUnicode(string(v))
	//	bb := []byte(ss)
	//	zlib.MyPrint(bb[0])
	//	nn  = append( bytecontent,bb[0])
	//}
	//zlib.ExitPrint(string(nn))
	//au = strings.Replace(au,`\\u`,`\u`,-1)

	//au := StrToUnicode(strconv.Itoa(actionMapT.Id))
	//zlib.MyPrint(au)
	//unicode.
	//content = au + content
	//content = strings.Replace(content,`\u`,`\\u`,-1)
	//mylog.Info("send content",content)
	//zlib.ExitPrint(3333)
	strId := strconv.Itoa(actionMapT.Id)
	content = BytesCombine([]byte(strId),content)
	//mylog.Info("send final content:",content)
	//zlib.MyPrint(content)
	wsConn,ok := wsConnManager.getConnPoolById(uid)
	if !ok {
		mylog.Error("wsConn not in pool,maybe del.")
		return
	}
	if wsConn.Status == CONN_STATUS_CLOSE {
		mylog.Error("wsConn status =CONN_STATUS_CLOSE.")
		return
	}
	//myMetrics.IncNode("output_num")
	//myMetrics.PlusNode("output_size",len(content))

	if action =="pushLogicFrame"{
		roomId := netWay.mySync.GetMySyncPlayerRoomById(uid)
		roomSyncMetrics := RoomSyncMetricsPool[roomId]
		roomSyncMetrics.OutputNum++
		roomSyncMetrics.OutputSize = roomSyncMetrics.OutputSize + len(content)
	}
	//wsConn.Conn.WriteMessage(websocket.TextMessage,[]byte(content))
	if mynetWay.Option.ContentType == CONTENT_TYPE_PROTOBUF {
		netWay.myWriteMessage(wsConn,websocket.BinaryMessage,content)
		//wsConn.Conn.WriteMessage(websocket.BinaryMessage,content)
	}else{
		netWay.myWriteMessage(wsConn,websocket.TextMessage,content)
		//wsConn.Conn.WriteMessage(websocket.TextMessage,content)
	}
}
func(netWay *NetWay) myWriteMessage(wsConn *WsConn ,msgCate int ,content []byte){
	defer func() {
		if err := recover(); err != nil {
			mylog.Error("wsConn.Conn.WriteMessage failed:",err)
			netWay.CloseOneConn(wsConn,CLOSE_SOURCE_SEND_MESSAGE)
		}
	}()
	wsConn.Conn.WriteMessage(msgCate,content)
}
//一个玩家取消了准备/报名
func(netWay *NetWay)cancelSign(requestCancelSign myproto.RequestPlayerMatchSignCancel,wsConn *WsConn){
	myMatch.delOnePlayer(wsConn.PlayerId)
}
//BytesCombine 多个[]byte数组合并成一个[]byte
func BytesCombine(pBytes ...[]byte) []byte {
	len := len(pBytes)
	s := make([][]byte, len)
	for index := 0; index < len; index++ {
		s[index] = pBytes[index]
	}
	sep := []byte("")
	return bytes.Join(s, sep)
}

func StrToUnicode(str string) (string) {
	DD := []rune(str)  //需要分割的字符串内容，将它转为字符，然后取长度。
	finallStr := ""
	for i := 0; i < len(DD); i++ {
		if unicode.Is(unicode.Scripts["Han"], DD[i]) {
			textQuoted := strconv.QuoteToASCII(string(DD[i]))
			finallStr += textQuoted[1 : len(textQuoted)-1]
		} else {
			h := fmt.Sprintf("%x",DD[i])
			finallStr += `\u` + isFullFour(h)
		}
	}
	return finallStr
}

func isFullFour(str string) (string) {
	if len(str) == 1 {
		str = "000" + str
	} else if len(str) == 2 {
		str = "00" + str
	} else if len(str) == 3 {
		str = "0" + str
	}
	return str
}

//给报名池添加用户，以快速成一局游戏
func(netWay *NetWay)playerMatchSign(requestPlayerMatchSign myproto.RequestPlayerMatchSign,wsConn *WsConn) {
	err := myMatch.addOnePlayer(wsConn.PlayerId)
	if err != nil{
		mylog.Error("playerReady",err.Error())
		return
	}
}
//监听到某个FD被关闭后，回调函数
func  (wsConn *WsConn)CloseHandler(code int, text string) error{
	mynetWay.CloseOneConn(wsConn, CLOSE_SOURCE_CLIENT)
	return nil
}

func (netWay *NetWay)CloseOneConn(wsConn *WsConn,source int){
	netWay.Option.Mylog.Info("wsConn close ,source : ",source,wsConn.PlayerId)
	if wsConn.Status == CONN_STATUS_EXECING {
		//把后台守护  协程 先关了
		wsConn.Status = CONN_STATUS_CLOSE
		wsConn.CloseChan <- 1
	}else{
		netWay.Option.Mylog.Error("wsConn.Status error")
		return
	}
	//netWay.Players.delById(wsConn.PlayerId)//这个不能删除，用于玩家掉线恢复的记录
	//先把玩家的在线状态给变更下，不然 mySync.close 里面获取房间在线人数，会有问题
	netWay.Players.upPlayerStatus(wsConn.PlayerId, PLAYER_STATUS_OFFLINE)
	//通知同步服务，先做构造处理
	netWay.mySync.Close(wsConn)

	err := wsConn.Conn.Close()
	netWay.Option.Mylog.Info("wsConn.Conn.Close err:",err)

	wsConnManager.delConnPool(wsConn.PlayerId)

	//处理掉-已报名的玩家
	myMatch.delOnePlayer(wsConn.PlayerId)
	//mySleepSecond(2,"CloseOneConn")
	myMetrics.input <- MetricsChanMsg{Key: "total.FDNum",Opt: 4}
	myMetrics.input <- MetricsChanMsg{Key: "historyFDDestroy",Opt: 2}
}
//退出
func  (netWay *NetWay)Quit() {
	ctx, _ := context.WithCancel(netWay.Option.Cxt)
	netWay.httpServer.Shutdown(ctx)

	netWay.MyCtxCancel() //关闭 FD超时检测 、玩家匹配成团

	if len(wsConnManager.Pool) == 0 {
		netWay.Option.Mylog.Warning("ConnPool is 0")
		return
	}
	for _, v := range wsConnManager.Pool {
		netWay.CloseOneConn(v, CLOSE_SOURCE_SIGNAL_QUIT)
	}
	netWay.Option.Mylog.Warning("quit finish")
}

//func (sync *Sync)checkRoomTimeoutLoop(ctx context.Context){
//	for{
//		select {
//		case <- ctx.Done():
//			mylog.Warning("checkRoomTimeoutLoop done.")
//			return
//		default:
//			sync.checkRoomTimeout()
//			time.Sleep(1 * time.Second)
//		}
//	}
//}
//func (sync *Sync)checkRoomTimeout(){
//	roomLen := len(mySyncRoomPool)
//	if roomLen <= 0{
//		return
//	}
//
//	now := int32( zlib.GetNowTimeSecondToInt())
//	for _,v:=range mySyncRoomPool{
//		if v.Status != ROOM_STATUS_EXECING{
//			continue
//		}
//
//		if now > v.Timeout{
//			sync.roomEnd(v.Id)
//		}
//	}
//}

