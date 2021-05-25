package netway

import (
	"context"
	"flag"
	"frame_sync/myproto"
	"frame_sync/myprotocol"
	"github.com/gorilla/websocket"
	"net/http"
	"zlib"
)

type NetWay struct {
	Option          	NetWayOption
	mySync 				*Sync
	httpServer      	*http.Server
	MyCtx           	context.Context
	MyCtxCancel     	context.CancelFunc
	PlayerManager         	*PlayerManager
	ProtocolActions 	*myprotocol.ProtocolActions
	CloseChan       	chan int32
	MatchSuccessChan    chan *Room
}

type NetWayOption struct {
	Host 				string		`json:"host"`
	Port 				string		`json:"port"`
	Mylog 				*zlib.Log	`json:"mylog"`
	Protocol 			int32		`json:"myprotocol"`		//协议  ，ws sockdt udp
	WsUri				string		`json:"wsUri"`			//接HOST的后面的URL地址
	ContentType 		int32		`json:"contentType"`	//json protobuf

	LoginAuthType		string		`json:"loginAuthType"`	//jwt
	LoginAuthSecretKey	string								//密钥

	MaxClientConnNum	int32		`json:"maxClientConnMum"`//客户端最大连接数
	MsgContentMax		int32								//一条消息内容最大值
	IOTimeout			int64								//read write sock fd 超时时间
	Cxt 				context.Context						//调用方的CTX，用于所有协程的退出操作
	MainChan			chan int32	`json:"-"`
	ConnTimeout 		int32		`json:"connTimeout"`						//检测FD最后更新时间

	MapSize				int32		`json:"mapSize"`		//地址大小，给前端初始化使用
	RoomPeople			int32		`json:"roomPeople"`		//一局游戏包含几个玩家
	RoomTimeout 		int32 		`json:"roomTimeout"`	//一个房间超时时间
	OffLineWaitTime		int32		`json:"offLineWaitTime"`//lockStep 玩家掉线后，其它玩家等待最长时间

	LockMode  			int32 		`json:"lockMode"`		//锁模式，乐观|悲观
	FPS 				int32 		`json:"fps"`			//frame pre second
	RoomReadyTimeout	int32		`json:"roomReadyTimeout"`//一个房间的，玩家的准备，超时时间

	Store 				int32 		`json:"store"`			//持久化：players room
}
//网络间的通信内容，最终被转换成的结构体
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
//var mySync 		*Sync
var mynetWay	*NetWay
var myMatch		*Match
var wsConnManager *WsConnManager
var myMetrics *Metrics
var mylog *zlib.Log
func NewNetWay(option NetWayOption)*NetWay {
	option.Mylog.Info("New NetWay instance :")
	zlib.PrintStruct(option," : ")

	netWay := new(NetWay)
	netWay.Option = option
	mynetWay = netWay
	//日志
	mylog = option.Mylog
	//匹配
	matchOption := MatchOption{
		RoomPeople: option.RoomPeople,
	}
	myMatch = NewMatch(matchOption)
	//同步
	syncOptions := Options{
		Log: mylog,
		FPS: option.FPS,
		MapSize: option.MapSize,
		LockMode : option.LockMode,
	}
	netWay.mySync = NewSync(syncOptions)
	//ws conn 管理
	wsConnManager = NewWsConnManager()
	//玩家信息管理模块
	netWay.PlayerManager = PlayerManagerNew()
	//传输内容体的配置
	netWay.ProtocolActions = myprotocol.ProtocolActionsNew()
	//统计模块
	myMetrics = MetricsNew()

	return netWay
}
//启动 - 入口
func (netWay *NetWay)Startup(){
	//启动时间
	s_time := zlib.GetNowMillisecond()
	myMetrics.fastLog("starupTime",METRICS_OPT_PLUS,int(s_time))
	//从外层调用的CTX上，派生netway自己的根ctx
	startupCtx ,cancel := context.WithCancel(netWay.Option.Cxt)
	netWay.MyCtx = startupCtx
	netWay.MyCtxCancel = cancel
	//匹配模块，成功匹配一组玩家后，推送消息的管道
	netWay.MatchSuccessChan = make(chan *Room)
	//开启匹配服务
	go myMatch.doingAndCreateRoom  (startupCtx,netWay.MatchSuccessChan)
	//监听超时的WS连接
	go wsConnManager.checkConnPoolTimeout(startupCtx)
	//接收<匹配成功>的房间信息，并分发
	go netWay.recviceMatchSuccess(startupCtx)
	//统计模块，消息监听开启
	go myMetrics.start(startupCtx)
	//玩家缓存状态，下线后，超时清理
	go netWay.PlayerManager.checkOfflineTimeout(startupCtx)
	//开始HTTP 监听 模块
	go netWay.startHttpServer()

	netWay.CloseChan = make(chan int32)
	go func() {
		<- netWay.CloseChan
		netWay.Quit()
	}()
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
		NewWsConn.Write([]byte(err.Error()),websocket.TextMessage)
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
		netWay.SendMsgCompressByUid(jwtData.Payload.Uid,"loginRes",&loginRes)
		netWay.CloseOneConn(NewWsConn, CLOSE_SOURCE_OVERRIDE)
		return
	}
	//给用户再绑定到 用户状态池,该池与连接池的区分 是：连接一但关闭，该元素即删除~而用户状态得需要保存
	playerConnInfo ,_ :=netWay.PlayerManager.addPlayer(jwtData.Payload.Uid)
	loginRes = myproto.ResponseLoginRes{
		Code: 200,
		ErrMsg: "",
		Player: &playerConnInfo,
	}
	//告知玩家：登陆结果
	netWay.SendMsgCompressByUid(jwtData.Payload.Uid,"loginRes",&loginRes)
	//统计 当前FD 数量/历史FD数量
	myMetrics.fastLog("total.fd.num",METRICS_OPT_INC,0)
	myMetrics.fastLog("history.create.fd.ok",METRICS_OPT_INC,0)
	//初始化即登陆成功的响应均完成后，开始该连接的 读取消息 协程
	go NewWsConn.IOLoop()
	//netWay.serverPingRtt(time.Duration(rttMinTimeSecond),NewWsConn,1)
	mylog.Info("wsHandler end ,player login success!!!")

}

func(netWay *NetWay)recviceMatchSuccess(ctx context.Context){
	mylog.Info("recviceMatchSuccess start:")
	isBreak := 0
	for{
		select {
			case newRoom :=  <-netWay.MatchSuccessChan:
				err := netWay.mySync.AddPoolElement(newRoom)
				if err !=nil{
					//responsePlayerMatchingFailed := myproto.ResponsePlayerMatchingFailed{
					//	RoomId: newRoom.Id,
					//	Msg: err.Error(),
					//}
					//mynetWay.SendMsgCompressByUid()
				}else{
					netWay.mySync.Start(newRoom.Id)
				}
			case   <-ctx.Done():
				isBreak = 1
			//default:
			//	time.Sleep(time.Second * 1)
			//mySleepSecond(1,"checkConnPoolTimeout")
		}
		if isBreak == 1{
			break
		}
	}
	mylog.Warning("recviceMatchSuccessone close")
}

func(netWay *NetWay)heartbeat(requestClientHeartbeat myproto.RequestClientHeartbeat,wsConn *WsConn){
	now := zlib.GetNowTimeSecondToInt()
	wsConn.UpTime = int32(now)
}
//=================================
//监听到某个FD被关闭后，回调函数
func  (wsConn *WsConn)CloseHandler(code int, text string) error{
	mynetWay.CloseOneConn(wsConn, CLOSE_SOURCE_CLIENT)
	return nil
}

func (netWay *NetWay)CloseOneConn(wsConn *WsConn,source int){
	netWay.Option.Mylog.Info("wsConn close ,source : ",source,wsConn.PlayerId)
	if wsConn.Status == CONN_STATUS_CLOSE {
		netWay.Option.Mylog.Error("wsConn.Status error")
		return
	}
	//把后台守护  协程 先关了
	wsConn.Status = CONN_STATUS_CLOSE
	wsConn.CloseChan <- 1
	//netWay.Players.delById(wsConn.PlayerId)//这个不能删除，用于玩家掉线恢复的记录
	//先把玩家的在线状态给变更下，不然 mySync.close 里面获取房间在线人数，会有问题
	netWay.PlayerManager.upPlayerStatus(wsConn.PlayerId, PLAYER_STATUS_OFFLINE)
	//通知同步服务，先做构造处理
	netWay.mySync.Close(wsConn)

	err := wsConn.Conn.Close()
	netWay.Option.Mylog.Info("wsConn.Conn.Close err:",err)

	wsConnManager.delConnPool(wsConn.PlayerId)

	//处理掉-已报名的玩家
	myMatch.realDelOnePlayer(wsConn.PlayerId)
	//mySleepSecond(2,"CloseOneConn")
	myMetrics.fastLog("total.fd.num",METRICS_OPT_DIM,0)
	myMetrics.fastLog("history.fd.destroy",METRICS_OPT_INC,0)
}
//退出
func  (netWay *NetWay)Quit() {
	ctx, _ := context.WithCancel(netWay.Option.Cxt)
	netWay.httpServer.Shutdown(ctx)

	netWay.MyCtxCancel() //关闭 所有  startup开启的协程

	if len(wsConnManager.Pool) == 0 {
		netWay.Option.Mylog.Warning("ConnPool is 0")
		return
	}
	for _, v := range wsConnManager.Pool {
		netWay.CloseOneConn(v, CLOSE_SOURCE_SIGNAL_QUIT)
	}
	netWay.Option.Mylog.Warning("quit finish")
}

//func StrToUnicode(str string) (string) {
//	DD := []rune(str)  //需要分割的字符串内容，将它转为字符，然后取长度。
//	finallStr := ""
//	for i := 0; i < len(DD); i++ {
//		if unicode.Is(unicode.Scripts["Han"], DD[i]) {
//			textQuoted := strconv.QuoteToASCII(string(DD[i]))
//			finallStr += textQuoted[1 : len(textQuoted)-1]
//		} else {
//			h := fmt.Sprintf("%x",DD[i])
//			finallStr += `\u` + isFullFour(h)
//		}
//	}
//	return finallStr
//}
//
//func isFullFour(str string) (string) {
//	if len(str) == 1 {
//		str = "000" + str
//	} else if len(str) == 2 {
//		str = "00" + str
//	} else if len(str) == 3 {
//		str = "0" + str
//	}
//	return str
//}