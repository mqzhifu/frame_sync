package netway

import (
	"context"
	"frame_sync/myproto"
	"frame_sync/myprotocol"
	"github.com/gorilla/websocket"
	"runtime"
	"strconv"
	"sync"
	"zlib"
)

type NetWay struct {
	MyCancelCtx           	context.Context
	MyCancelFunc		func()
	CloseChan       	chan int32
	MatchSuccessChan    chan *Room
	Status 				int

	Option          	NetWayOption
}

type NetWayOption struct {
	ListenIp			string		`json:"listenIp"`		//程序启动时监听的IP
	OutIp				string		`json:"outIp"`			//对外访问的IP
	HttpdRootPath 		string 		`json:"httpdRootPath"`
	HttpPort 			string 		`json:"httpPort"`		//短连接端口号
	WsPort 				string		`json:"wsPort"`			//监听端口号
	TcpPort 			string		`json:"tcpPort"`		//监听端口号
	UdpPort				string 		`json:"udpPort"`		//UDP端口号
	Mylog 				*zlib.Log	`json:"-"`
	Protocol 			int32		`json:"protocol"`		//默认协议：ws tcp udp
	WsUri				string		`json:"wsUri"`			//接HOST的后面的URL地址
	ContentType 		int32		`json:"contentType"`	//默认内容格式 ：json protobuf

	LoginAuthType		string		`json:"loginAuthType"`	//jwt
	LoginAuthSecretKey	string								//密钥

	MaxClientConnNum	int32		`json:"maxClientConnMum"`//客户端最大连接数
	MsgContentMax		int32								//一条消息内容最大值
	IOTimeout			int64								//read write sock fd 超时时间
	OutCxt 				context.Context `-`					//调用方的CTX，用于所有协程的退出操作
	MainChan			chan int32	`json:"-"`
	ConnTimeout 		int32		`json:"connTimeout"`	//检测FD最后更新时间

	MapSize				int32		`json:"mapSize"`		//地址大小，给前端初始化使用
	RoomPeople			int32		`json:"roomPeople"`		//一局游戏包含几个玩家
	RoomTimeout 		int32 		`json:"roomTimeout"`	//一个房间超时时间
	OffLineWaitTime		int32		`json:"offLineWaitTime"`//lockStep 玩家掉线后，其它玩家等待最长时间

	LockMode  			int32 		`json:"lockMode"`		//锁模式，乐观|悲观
	FPS 				int32 		`json:"fps"`			//frame pre second
	RoomReadyTimeout	int32		`json:"roomReadyTimeout"`//一个房间的，玩家的准备，超时时间

	Store 				int32 		`json:"store"`			//持久化：players room
	LogOption 			zlib.LogOption `json:"-"`
	OutCancelFunc		context.CancelFunc `json:"-"`
}

//下面是全局变量，主要是快捷方便使用，没实际意义
var myNetWay	*NetWay

var myMatch		*Match
var myMetrics 	*Metrics
var mySync		*Sync
var myProtocolActions *myprotocol.ProtocolActions
var myProtocolManager *ProtocolManager
var myPlayerManager  *PlayerManager
var myHttpd 	*Httpd
var connManager *ConnManager

var mylog		*zlib.Log

var RecoverGoRoutineRetryTimes  = make(map[string]int)
var RecoverGoRoutineRetryTimesRWLock = &sync.RWMutex{}

func NewNetWay(option NetWayOption)*NetWay {
	option.Mylog.Info("New NetWay instance :")
	option.Mylog.Debug("option:",option)

	netWay := new(NetWay)
	netWay.Option = option
	netWay.Status = NETWAY_STATUS_INT

	myNetWay 	= netWay
	//日志
	mylog = option.Mylog
	//协议管理适配器
	protocolManagerOption := ProtocolManagerOption{
		Ip				: netWay.Option.ListenIp,
		HttpPort		: netWay.Option.WsPort,
		TcpPort			: netWay.Option.TcpPort,
		WsUri			: netWay.Option.WsUri,
		OpenNewConnBack	: netWay.OpenNewConn,
	}
	myProtocolManager =  NewProtocolManager(protocolManagerOption)
	//http 工具
	httpdOption := HttpdOption {
		LogOption 	: netWay.Option.LogOption,
		RootPath 	: netWay.Option.HttpdRootPath,
		Ip			: netWay.Option.ListenIp,
		Port		: netWay.Option.HttpPort,
		ParentCtx 	: option.OutCxt,
	}
	myHttpd = NewHttpd(httpdOption)
	//匹配
	//匹配模块，成功匹配一组玩家后，推送消息的管道
	netWay.MatchSuccessChan = make(chan *Room,10)
	matchOption := MatchOption{
		RoomPeople: option.RoomPeople,
		RoomReadyTimeout: option.RoomReadyTimeout,
		MatchSuccessChan :netWay.MatchSuccessChan,
	}
	myMatch = NewMatch(matchOption)
	//同步
	syncOptions := SyncOption{
		Log			: mylog,
		FPS			: option.FPS,
		MapSize		: option.MapSize,
		LockMode 	: option.LockMode,
	}
	mySync = NewSync(syncOptions)
	//ws conn 管理
	connManager = NewConnManager(option.MaxClientConnNum,option.ConnTimeout)
	//玩家信息管理模块
	playerManagerOption := PlayerManagerOption{
		Log: mylog,
		store: option.Store,
		ContentType: option.ContentType,
		Protocol: option.Protocol,
	}
	myPlayerManager = NewPlayerManager(playerManagerOption)
	//传输内容体的配置
	myProtocolActions = myprotocol.NewProtocolActions(mylog)
	//统计模块
	myMetrics = NewMetrics()

	//在外层的CTX上，派生netway自己的根ctx
	//startupCtx ,cancel := context.WithCancel(netWay.Option.OutCxt)
	startupCtx , cancelFunc := context.WithCancel(netWay.Option.OutCxt)
	netWay.MyCancelCtx = startupCtx
	netWay.MyCancelFunc = cancelFunc
	return netWay
}
//启动 - 入口
func (netWay *NetWay)Startup(){
	//fd io timeout
	//fd connect timeout
	//room timeout
	//conn timeout
	//player timeout
	//RoomReadyTimeout
	//rtt timeout

	//connTimeout demon ->netWayClose ->sync.close
	//connFDClose
	//connFDException

	mylog.Alert("netWay Startup:")
	//启动时间
	startTime := zlib.GetNowMillisecond()
	myMetrics.fastLog("netWay starUpTime",METRICS_OPT_PLUS,int(startTime))
	netWay.Status = NETWAY_STATUS_START

	go myHttpd.start(netWay.MyCancelCtx)
	//开启匹配服务
	go myMatch.Start  (netWay.MyCancelCtx)
	//监听超时的WS连接
	go connManager.Start(netWay.MyCancelCtx)
	//接收<匹配成功>的房间信息，并分发
	go mySync.Start(netWay.MyCancelCtx)
	//统计模块，消息监听开启
	go myMetrics.start(netWay.MyCancelCtx)
	//玩家缓存状态，下线后，超时清理
	go myPlayerManager.Start(netWay.MyCancelCtx)
	myProtocolManager.Start(netWay.MyCancelCtx)

	netWay.CloseChan = make(chan int32)
	go netWay.listenQuit()
}
func(netWay *NetWay)listenQuit(){
	mylog.Alert(" listenQuit...")
	//这里是2种关闭方式
	//1. 内置的关闭管道接收外部直接发送的信号
	//2. 监听外部给context管道信号
	select {
		case <- netWay.CloseChan:
			netWay.Quit()
		//case <- netWay.Option.OutCxt.Done():
		case <- netWay.MyCancelCtx.Done():
			netWay.Quit()
	}
}
//一个客户端连接请求进入
func(netWay *NetWay)OpenNewConn( connFD FDAdapter) {
	mylog.Info("OpenNewConn:")
	if netWay.Status == NETWAY_STATUS_CLOSE{
		errMsg := "netWay closing... not accept new connect!"
		netWay.Option.Mylog.Error(errMsg)
		connFD.WriteMessage(websocket.TextMessage,[]byte(errMsg))
		connFD.Close()
		return
	}
	//创建一个连接元素，将WS FD 保存到该容器中
	NewConn ,err := connManager.CreateOneConn(connFD)
	if err !=nil{
		netWay.Option.Mylog.Error(err.Error())
		NewConn.Write([]byte(err.Error()),websocket.TextMessage)
		netWay.CloseOneConn(NewConn, CLOSE_SOURCE_CREATE)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			mylog.Panic("OpenNewConn:",err)
			netWay.CloseOneConn(NewConn, CLOSE_SOURCE_OPEN_PANIC)
		}
	}()
	//登陆验证
	jwtData,err,firstMsg := netWay.loginPre(NewConn)
	if err != nil{
		return
	}
	var loginRes myproto.ResponseLoginRes
	//登陆验证通过，开始添加各种状态及初始化
	NewConn.PlayerId = jwtData.Payload.Uid
	//将新的连接加入到连接池中，并且与玩家ID绑定
	err = connManager.addConnPool( NewConn)
	if err != nil{
		loginRes = myproto.ResponseLoginRes{
			Code: 500,
			ErrMsg: err.Error(),
		}
		netWay.SendMsgCompressByUid(jwtData.Payload.Uid,"loginRes",&loginRes)
		netWay.CloseOneConn(NewConn, CLOSE_SOURCE_OVERRIDE)
		return
	}
	//给用户再绑定到 用户状态池,该池与连接池的区分 是：连接一但关闭，该元素即删除~而用户状态得需要保存
	playerConnInfo ,_ := myPlayerManager.addPlayer(jwtData.Payload.Uid,firstMsg)
	loginRes = myproto.ResponseLoginRes{
		Code: 200,
		ErrMsg: "",
		Player: &playerConnInfo,
	}
	NewConn.SessionId = playerConnInfo.SessionId
	//告知玩家：登陆结果
	netWay.SendMsgCompressByUid(jwtData.Payload.Uid,"loginRes",&loginRes)
	//统计 当前FD 数量/历史FD数量
	myMetrics.fastLog("total.fd.num",METRICS_OPT_INC,0)
	myMetrics.fastLog("history.create.fd.ok",METRICS_OPT_INC,0)
	//初始化即登陆成功的响应均完成后，开始该连接的 读取消息 协程
	go NewConn.IOLoop()
	//netWay.serverPingRtt(time.Duration(rttMinTimeSecond),NewWsConn,1)
	mylog.Info("wsHandler end ,player login success!!!")

}

func(netWay *NetWay)heartbeat(requestClientHeartbeat myproto.RequestClientHeartbeat,conn *Conn){
	now := zlib.GetNowTimeSecondToInt()
	conn.UpTime = int32(now)
}
//=================================
func (netWay *NetWay)CloseOneConn(conn *Conn,source int){
	mylog.Info("Conn close ,source : ",source,conn.PlayerId)
	if conn.Status == CONN_STATUS_CLOSE {
		netWay.Option.Mylog.Error("CloseOneConn error :Conn.Status == CLOSE")
		return
	}
	//通知同步服务，先做构造处理
	mySync.CloseOne(conn)//这里可能还要再发消息

	//状态更新为已关闭，防止重复关闭
	conn.Status = CONN_STATUS_CLOSE
	//把后台守护  协程 先关了，不再收消息了
	conn.CloseChan <- 1
	//netWay.Players.delById(Conn.PlayerId)//这个不能删除，用于玩家掉线恢复的记录
	//先把玩家的在线状态给变更下，不然 mySync.close 里面获取房间在线人数，会有问题
	myPlayerManager.upPlayerStatus(conn.PlayerId, PLAYER_STATUS_OFFLINE)
	err := conn.Conn.Close()
	if err != nil{
		netWay.Option.Mylog.Error("Conn.Conn.Close err:",err)
	}

	connManager.delConnPool(conn.PlayerId)
	//处理掉-已报名的玩家
	myMatch.realDelOnePlayer(conn.PlayerId)
	//mySleepSecond(2,"CloseOneConn")
	myMetrics.fastLog("total.fd.num",METRICS_OPT_DIM,0)
	myMetrics.fastLog("history.fd.destroy",METRICS_OPT_INC,0)
}
//退出，目前能直接调用此函数的，就只有一种情况：
//MAIN 接收到了中断信号，并执行了：context-cancel()，然后，startup函数的守护监听到，调用些方法
func  (netWay *NetWay)Quit() {
	mylog.Warning("netWay.Quit")
	if netWay.Status == NETWAY_STATUS_CLOSE{
		mylog.Error("Quit err :netWay.Status ==  NETWAY_STATUS_CLOSE")
		return
	}
	netWay.Status = NETWAY_STATUS_CLOSE//更新状态为：关闭

	myHttpd.shutdown()
	myMatch.Shutdown()
	mySync.Shutdown()
	myPlayerManager.Shutdown()
	connManager.Shutdown()
	myProtocolManager.Shutdown()
	myMetrics.Shutdown()
	//go netWay.PlayerManager.checkOfflineTimeout(startupCtx)
	netWay.Option.OutCancelFunc()
}


func  (netWay *NetWay)RecoverGoRoutine(back func(ctx context.Context),ctx context.Context,err interface{}){
	pc, file, lineNo, ok := runtime.Caller(3)
	if !ok {
		mylog.Error("runtime.Caller ok is false :",ok)
	}
	funcName := runtime.FuncForPC(pc).Name()
	mylog.Info(" RecoverGoRoutine  panic in defer  :"+ funcName + " "+file + " "+ strconv.Itoa(lineNo))
	RecoverGoRoutineRetryTimesRWLock.RLock()
	retryTimes , ok := RecoverGoRoutineRetryTimes[funcName]
	RecoverGoRoutineRetryTimesRWLock.RUnlock()
	if ok{
		if retryTimes > 3{
			mylog.Error("retry than max times")
			panic(err)
			return
		}else{
			RecoverGoRoutineRetryTimesRWLock.Lock()
			RecoverGoRoutineRetryTimes[funcName]++
			RecoverGoRoutineRetryTimesRWLock.Unlock()
			mylog.Info("RecoverGoRoutineRetryTimes = ",RecoverGoRoutineRetryTimes[funcName])
		}
	}else{
		mylog.Info("RecoverGoRoutineRetryTimes = 1")
		RecoverGoRoutineRetryTimesRWLock.Lock()
		RecoverGoRoutineRetryTimes[funcName] = 1
		RecoverGoRoutineRetryTimesRWLock.Unlock()
	}
	go back(ctx)
}