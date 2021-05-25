package main

import (
	"context"
	"frame_sync/netway"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"zlib"
)
func tt1(m *sync.Mutex){
	m.Lock()
	return
}
func test(){

}
var mylog *zlib.Log
func main(){
	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG
	test()

	if len(os.Args) < 4{
		msg := "os.Args len < 4 , ex :  env=dev , ip=127.0.0.1 , port=2222 , log_base_path=/data/www/golang/src/logs"
		zlib.ExitPrint(msg)
	}

	env 			:= os.Args[1]
	ip 				:= os.Args[2]
	port 			:= os.Args[3]
	log_base_path 	:= os.Args[4]

	msg := "os.Args: env= "+env +" ,ip= "+ip +" ,port=" + port+ " ,log_base_path= " +log_base_path
	zlib.MyPrint(msg)
	if !zlib.CheckEnvExist(env){
		list := zlib.GetEnvList()
		zlib.ExitPrint("env is err , list:",list)
	}

	enter(log_base_path,ip,port)
}

func enter(log_base_path string,ip string ,port string){
	rootCtx := context.Background()

	logOption := zlib.LogOption{
		OutFilePath : log_base_path,
		OutFileName: "frame_sync.log",
		Level : 511,
		Target : 1,
	}
	newlog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint("new log err",errs.Error())
	}
	mylog = newlog
	//mainChan := make(chan int )
	newNetWayOption := netway.NetWayOption{
		//Host 				:"192.168.192.125",
		//Port 				:"2222",
		Mylog 				:mylog,
		Host				:ip,
		Port				:port,
		//ContentType			:netway.CONTENT_TYPE_JSON,
		ContentType			:netway.CONTENT_TYPE_PROTOBUF,
		LoginAuthType		:"jwt",
		LoginAuthSecretKey	:"chukong",
		IOTimeout			:3,
		Cxt 				:rootCtx,
		ConnTimeout			: 60,
		Protocol			: netway.PROTOCOL_WEBSOCKET,
		WsUri				: "/ws",
		MaxClientConnNum	:65535,
		RoomPeople			:2,
		//RoomTimeout 		:120,
		RoomReadyTimeout 	:10,
		OffLineWaitTime		:20,//玩家掉线后，等待多久
		MapSize				:10,
		LockMode			: netway.LOCK_MODE_PESSIMISTIC,
		FPS					:10,
		Store				: 0,
	}
	//创建网关，并启动
	newNetWay := netway.NewNetWay(newNetWayOption)
	go newNetWay.Startup()
	//创建main信号
	mainCtx,mainCancel := context.WithCancel(rootCtx)
	//开启信号监听
	go DemonSignal(newNetWay,mainCtx,mainCancel)
	//阻塞，如果有信号终止了，最后要把main正常结束
	<-mainCtx.Done()
	mylog.CloseChan <- 1
	mylog.Warning("main end...")
	//这里做个容错，可能会遗漏掉的协程未结束 或 结束程序有点慢
	time.Sleep(1 * time.Second)
}
//信号 处理
func  DemonSignal(newNetWay *netway.NetWay,mainCtx context.Context,mainCancel context.CancelFunc){
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
				newNetWay.Quit()
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
	mylog.Warning("DemonSignal end")
	mainCancel()
}

//睡眠 - 协程
func   mySleepSecond(second time.Duration , msg string){
	//mylog.Info(msg," sleep second ", strconv.Itoa(int(second)))
	time.Sleep(second * time.Second)
}