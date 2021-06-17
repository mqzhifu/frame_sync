package main

import (
	"context"
	"frame_sync/netway"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"zlib"
)

type CmdArgs struct {
	Env 			string	`seq:"1" err:"env=local"`
	Ip 				string	`seq:"2" err:"ip=127.0.0.1"`
	HttpPort 		string	`seq:"3" err:"WsPort=2222"`
	WsPort 			string	`seq:"4" err:"WsPort=2223"`
	TcpPort 		string	`seq:"5" err:"TcpPort=2224"`
	LogBasePath 	string	`seq:"6" err:"log_base_path=/data/www/golang/src/logs cs=server"`
	ClientServer 	string 	`seq:"7" err:"cs=serve"`
}
//全局LOG类，快捷调用
var mainlog *zlib.Log
var mainOutPrefix = "main root :"
func main(){
	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG

	cmdArgsStruct := CmdArgs{}
	cmsArg ,err := zlib.CmsArgs(cmdArgsStruct)
	if err != nil{
		zlib.ExitPrint(mainOutPrefix + " err " +err.Error())
	}
	zlib.MyPrint(mainOutPrefix + " argc  ")
	for k,v := range cmsArg{
		msg :=  k + ":"+ v
		zlib.MyPrint(msg)
	}


	if !zlib.CheckEnvExist(cmsArg["Env"]){
		list := zlib.GetEnvList()
		zlib.ExitPrint(mainOutPrefix + "env is err , list:",list)
	}

	enter(cmsArg)
}

func enter(cmsArg map[string]string){
	appId := 2
	appM  := zlib.NewAppManager()
	app,empty := appM.GetById(appId)
	if empty{
		zlib.ExitPrint(mainOutPrefix + " err " + ": appId is empty "+ strconv.Itoa(appId))
	}
	zlib.MyPrint(mainOutPrefix + " appInfo  ")
	zlib.PrintStruct(app," : ")

	//创建一个根-空 ctx
	rootBackgroundCtx := context.Background()
	//继承上面的根CTX，合建一个 cancel ctx ，后续所有的协程均要从此CTX继承
	rootCancelCtx,rootCancelFunc  := context.WithCancel(rootBackgroundCtx)

	logOutFilePath :=  zlib.BasePathPlusTypeStr(cmsArg["LogBasePath"],appM.GetTypeName(app.Type))
	logOption := zlib.LogOption{
		AppId			: app.Id,
		ModuleId		: 1,
		OutFilePath 	: logOutFilePath,
		OutFileFileName	: app.Key,
		Level 			: zlib.LEVEL_ALL,
		OutTarget 		: zlib.OUT_TARGET_ALL,
		OutContentType	: zlib.CONTENT_TYPE_JSON,
		OutFileHashType	: zlib.FILE_HASH_DAY,
	}
	newlog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint(mainOutPrefix + "new log err",errs.Error())
	}
	mainlog = newlog
	newNetWayOption := netway.NetWayOption{
		Mylog 				:mainlog,
		ListenIp			:cmsArg["Ip"],
		OutIp				:cmsArg["Ip"],
		//OutIp				:"39.106.65.76",
		HttpPort			:cmsArg["HttpPort"],
		WsPort				:cmsArg["WsPort"],
		TcpPort				:cmsArg["TcpPort"],
		UdpPort				:"9999",
		//ContentType		:netway.CONTENT_TYPE_JSON,
		ContentType			:netway.CONTENT_TYPE_PROTOBUF,
		LoginAuthType		:"jwt",
		LoginAuthSecretKey	:"chukong",
		IOTimeout			:3,
		OutCxt 				:rootCancelCtx,
		ConnTimeout			:60,
		//Protocol			:netway.PROTOCOL_TCP,
		Protocol			:netway.PROTOCOL_WEBSOCKET,
		WsUri				:"/ws",
		MaxClientConnNum	:65535,
		RoomPeople			:2,
		RoomReadyTimeout 	:10,
		OffLineWaitTime		:20,//玩家掉线后，等待多久
		MapSize				:10,
		LockMode			:netway.LOCK_MODE_PESSIMISTIC,
		FPS					:10,
		Store				:0,
		LogOption: logOption,
	}
	//测试使用，开始TCP/UDP client端
	testSwitchClientServer(cmsArg["ClientServer"],newNetWayOption)
	//创建网关，并启动
	newNetWay := netway.NewNetWay(newNetWayOption)
	go newNetWay.Startup()
	//创建main信号
	//mainCtx,mainCancel := context.WithCancel(rootCtx)
	//开启信号监听
	go DemonSignal(newNetWay,rootCancelFunc)
	//阻塞main主线程，停止的话，只有一种可能：接收到了信号
	<-rootCancelCtx.Done()
	//这里做个容错，可能会遗漏掉的协程未结束 或 结束程序有点慢
	time.Sleep(1 * time.Second)

	mainlog.CloseChan <- 1
	time.Sleep(500 * time.Millisecond)
	mainlog.Warning("main end...")
}
//信号 处理
func  DemonSignal(newNetWay *netway.NetWay,mainCancelFuc context.CancelFunc){
	mainlog.Alert("SIGNAL start : ")
	c := make(chan os.Signal)
	//syscall.SIGHUP :ssh 挂断会造成这个信号被捕获，先注释掉吧
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	prefix := "SIGNAL-DEMON :"
	for{
		sign := <- c
		mainlog.Warning(prefix,sign)
		switch sign {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				mainlog.Warning(prefix+" exit!!!")
				//newNetWay.CloseChan <- 1
				goto end
			case syscall.SIGUSR1:
				mainlog.Warning(prefix+" usr1!!!")
			case syscall.SIGUSR2:
				mainlog.Warning(prefix+" usr2!!!")
			default:
				mainlog.Warning(prefix+" unknow!!!")
		}
		mySleepSecond(1,prefix)
	}
end :
	mainlog.Alert(netway.CTX_DONE_PRE + " DemonSignal end ,exec mainCancelFuc()")
	mainCancelFuc()
}

//睡眠 - 协程
func   mySleepSecond(second time.Duration , msg string){
	//mylog.Info(msg," sleep second ", strconv.Itoa(int(second)))
	time.Sleep(second * time.Second)
}

//测试使用，开始TCP/UDP client端
func testSwitchClientServer(clientServer string,newNetWayOption netway.NetWayOption ){
	switch clientServer {
	case "client":
		netway.StartTcpClient(newNetWayOption,mainlog)
		cc := make(chan int)
		<- cc
		zlib.ExitPrint(1111111111)
	case "udpClient":
		udpServer :=  netway.UdpServerNew(newNetWayOption,mainlog)
		udpServer.StartClient()
		cc := make(chan int)
		<- cc
		zlib.ExitPrint(22222)
	case "udpServer":
		udpServer :=  netway.UdpServerNew(newNetWayOption,mainlog)
		udpServer.Start()
		cc := make(chan int)
		<- cc
		zlib.ExitPrint(33333)
	}
}