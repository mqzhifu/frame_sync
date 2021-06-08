package main

import (
	"context"
	"frame_sync/netway"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zlib"
)

func tc (c chan int)<- chan int{
	return c
}

func testChan(){
	c:= make(chan int)
	go func(c1 chan int ){
		xx := <- c1
		zlib.MyPrint(xx," exit 1")
	}(c)

	go func(c2 chan int ){
		xx := <- c2
		zlib.MyPrint(xx," exit 2")
	}(c)

	time.Sleep(time.Second * 1)

	close(c)
	time.Sleep(time.Second * 1)
	zlib.ExitPrint(111)
}

func testContext(){
	//context.WithValue(bg,"a","b")



	testChan()




	bg := context.Background()
	bgChildCtx,bgChildCancel := context.WithCancel(bg)
	go func(myBgChildCtx context.Context ){

		xx := <- myBgChildCtx.Done()
		zlib.MyPrint(xx," exit 1")
	}(bgChildCtx)


	bgChildChildCtx,bgChildChildCancel := context.WithCancel(bgChildCtx)

	go func(mybgChildChildCtx context.Context ){

		xx := <- mybgChildChildCtx.Done()
		zlib.MyPrint(xx," exit 2")
	}(bgChildChildCtx)

	time.Sleep(time.Second * 1)
	bgChildChildCancel()
	time.Sleep(time.Second * 1)
	zlib.ExitPrint(3333)
	bgChildCancel()
}
//创建唯一ID

var mylog *zlib.Log
func main(){
	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG

	type CmdArgs struct {
		Env 			string	`seq:"1" err:"env=dev"`
		Ip 				string	`seq:"2" err:"ip=127.0.0.1"`
		WsPort 			string	`seq:"3" err:"WsPort=2222"`
		TcpPort 		string	`seq:"4" err:"TcpPort=2223"`
		LogBasePath 	string	`seq:"5" err:"log_base_path=/data/www/golang/src/logs cs=server"`
		ClientServer 	string 	`seq:"6" err:"cs=serve"`
	}
	cmdArgsStruct := CmdArgs{}
	cmsArg ,err := zlib.CmsArgs(cmdArgsStruct)
	if err != nil{
		zlib.ExitPrint(err.Error())
	}
	msg := "argc : "
	for k,v := range cmsArg{
		msg +=  k + ":"+ v + " , "
	}
	zlib.MyPrint(msg)

	if !zlib.CheckEnvExist(cmsArg["Env"]){
		list := zlib.GetEnvList()
		zlib.ExitPrint("env is err , list:",list)
	}

	//testContext()
	enter(cmsArg)
}

func enter(cmsArg map[string]string){
	log_base_path := cmsArg["LogBasePath"]
	ip := cmsArg["Ip"]
	wsPort:= cmsArg["WsPort"]
	tcpPort:= cmsArg["TcpPort"]
	ClientServer:= cmsArg["ClientServer"]
	//创建一个根-空 ctx
	rootBackgroundCtx := context.Background()
	//继承上面的根CTX，合建一个 cancel ctx ，后续所有的协程均要从此CTX继承
	rootCancelCtx,rootCancelFunc  := context.WithCancel(rootBackgroundCtx)
	logOption := zlib.LogOption{
		OutFilePath : log_base_path,
		OutFileName: "frame_sync.log",
		Level : 511,
		Target : 3,
		OutFileHashType: zlib.FILE_HASH_HOUR,

	}
	newlog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint("new log err",errs.Error())
	}
	mylog = newlog

	newNetWayOption := netway.NetWayOption{
		Mylog 				:mylog,
		ListenIp			:ip,
		OutIp				:ip,
		//OutIp				: "39.106.65.76",
		WsPort				:wsPort,
		TcpPort				:tcpPort,
		UdpPort				:"9999",
		//ContentType			:netway.CONTENT_TYPE_JSON,
		ContentType			:netway.CONTENT_TYPE_PROTOBUF,
		LoginAuthType		:"jwt",
		LoginAuthSecretKey	:"chukong",
		IOTimeout			:3,
		OutCxt 				:rootCancelCtx,
		ConnTimeout			: 60,
		//Protocol			: netway.PROTOCOL_TCP,
		Protocol			: netway.PROTOCOL_WEBSOCKET,
		WsUri				: "/ws",
		MaxClientConnNum	:65535,
		RoomPeople			:2,
		RoomReadyTimeout 	:10,
		OffLineWaitTime		:20,//玩家掉线后，等待多久
		MapSize				:10,
		LockMode			: netway.LOCK_MODE_PESSIMISTIC,
		FPS					:10,
		Store				: 0,
	}
	//测试使用，开始TCP/UDP client端
	testSwitchClientServer(ClientServer,newNetWayOption)
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

	mylog.CloseChan <- 1
	time.Sleep(500 * time.Millisecond)
	mylog.Warning("main end...")
}
//测试使用，开始TCP/UDP client端
func testSwitchClientServer(clientServer string,newNetWayOption netway.NetWayOption ){
	switch clientServer {
		case "client":
			netway.StartTcpClient(newNetWayOption,mylog)
			cc := make(chan int)
			<- cc
			zlib.ExitPrint(1111111111)
		case "udpClient":
			udpServer :=  netway.UdpServerNew(newNetWayOption,mylog)
			udpServer.StartClient()
			cc := make(chan int)
			<- cc
			zlib.ExitPrint(22222)
		case "udpServer":
			udpServer :=  netway.UdpServerNew(newNetWayOption,mylog)
			udpServer.Start()
			cc := make(chan int)
			<- cc
			zlib.ExitPrint(33333)
	}
}
//信号 处理
func  DemonSignal(newNetWay *netway.NetWay,mainCancelFuc context.CancelFunc){
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
				//newNetWay.CloseChan <- 1
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
	mylog.Alert(netway.CTX_DONE_PRE + " DemonSignal end")
	mainCancelFuc()
}

//睡眠 - 协程
func   mySleepSecond(second time.Duration , msg string){
	//mylog.Info(msg," sleep second ", strconv.Itoa(int(second)))
	time.Sleep(second * time.Second)
}