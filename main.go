package main

import (
	"container/list"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zlib"
)

//http://code.google.com/p/go.net/websocket
//http://code.google.com/p/go.net/websocket
//http://golang.org/x/net/websocket
//github.com/gorilla/websocket
//http://github.com/golang/net


func test(){
	queue := list.New()
	queue.PushBack("aaaa")
	element := queue.Front()
	info := element.Value.(string)
	zlib.ExitPrint(info)
}

var mylog *zlib.Log
func main(){
	//test()
	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG

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
	if !CheckEnvExist(env){
		list := GetEnvList()
		zlib.ExitPrint("env is err , list:",list)
	}


	rootCtx := context.Background()
	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG
	logOption := zlib.LogOption{
		OutFilePath : log_base_path,
		OutFileName: "frame_sync.log",
		Level : 511,
		Target : 7,
	}
	newlog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint("new log err",errs.Error())
	}
	mylog = newlog

	//mainChan := make(chan int )
	newNetWayOption := NetWayOption{
		//Host 				:"192.168.192.125",
		//Port 				:"2222",
		Mylog 				:mylog,
		Host				:ip,
		Port				:port,
		ContentType			:CONTENT_TYPE_JSON,
		LoginAuthType		:"jwt",
		LoginAuthSecretKey	:"chukong",
		IOTimeout			:3,
		Cxt 				:rootCtx,
		ConnTimeout			: 60,
		Protocol: PROTOCOL_WEBSOCKET,
		WsUri: "/ws",
		MaxClientConnNum	:65535,
		//MainChan			:mainChan,
		RoomPeople			:4,
		RoomTimeout 		:120,
		MapSize				:10,
		OffLineWaitTime		:20,
		LockMode: LOCK_MODE_PESSIMISTIC,
		FPS					:15,
	}
	newNetWay := NewNetWay(newNetWayOption)
	go newNetWay.Startup()

	mainCtx,mainCancel := context.WithCancel(rootCtx)
	go DemonSignal(newNetWay,mainCtx,mainCancel)
	for{
		select {
		case   <-mainCtx.Done():
			mylog.Warning("mainChan")
			goto mainEnd
		default:
			time.Sleep(time.Second * 1)
			//mySleepSecond(1, "main")
		}
	}

	mainEnd:
		mylog.Warning("main end...")

	time.Sleep(5 * time.Second)
}
//信号 处理
func  DemonSignal(newNetWay *NetWay,mainCtx context.Context,mainCancel context.CancelFunc){
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