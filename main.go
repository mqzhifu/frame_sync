package main

import (
	"context"
	"time"
	"zlib"
)

//http://code.google.com/p/go.net/websocket
//http://code.google.com/p/go.net/websocket
//http://golang.org/x/net/websocket
//github.com/gorilla/websocket
//http://github.com/golang/net

var mylog *zlib.Log
func main(){
	mainChan := make(chan int )

	rootCtx := context.Background()
	zlib.LogLevelFlag = zlib.LOG_LEVEL_DEBUG
	logOption := zlib.LogOption{
		OutFilePath : "/data/www/golang/src/logs",
		OutFileName: "frame_sync.log",
		Level : 511,
		Target : 7,
	}
	newlog,errs  := zlib.NewLog(logOption)
	if errs != nil{
		zlib.ExitPrint("new log err",errs.Error())
	}
	mylog = newlog


	newNetWayOption := NetWayOption{
		Mylog 				:mylog,
		Host 				:"127.0.0.1",
		Port 				:"2222",
		ContentType			:CONTENT_TYPE_JSON,
		LoginAuthType		:"jwt",
		LoginAuthSecretKey	:"chukong",
		IOTimeout			:3,
		Cxt 				:rootCtx,
		ConnTimeout			: 60,
		Protocol: PROTOCOL_WEBSOCKET,
		WsUri: "/ws",
		MaxClientConnNum	:65535,
		MainChan			:mainChan,

	}
	newNetWay := NewNetWay(newNetWayOption)
	go newNetWay.Startup()


	for{
		select {
		case a := <-mainChan:
			mylog.Warning("mainChan",a)
			goto eee
		default:
			time.Sleep(time.Second * 1000)
			//mySleepSecond(1, "main")
		}
	}

	eee:
		mylog.Warning("main end...")

	//http.Handle("/ws", websocket.Handler(wsHandler))
	//http.HandleFunc("/",wwwHandler)
	//mylog.Info("start demon /www...")
	//if err := http.ListenAndServe("127.0.0.1:1111", nil); err != nil {
	//	log.Fatal("ListenAndServe:", err)
	//}
}

////信号 处理
//func  DemonSignal(newNetWay *NetWay){
//	mylog.Warning("SIGNAL init : ")
//	c := make(chan os.Signal)
//	//syscall.SIGHUP :ssh 挂断会造成这个信号被捕获，先注释掉吧
//	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
//	prefix := "SIGNAL-DEMON :"
//	for{
//		sign := <- c
//		mylog.Warning(prefix,sign)
//		switch sign {
//		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
//			mylog.Warning(prefix+" exit!!!")
//			netWay.Quit()
//			goto end
//		case syscall.SIGUSR1:
//			mylog.Warning(prefix+" usr1!!!")
//		case syscall.SIGUSR2:
//			mylog.Warning(prefix+" usr2!!!")
//		default:
//			mylog.Warning(prefix+" unknow!!!")
//		}
//		mySleepSecond(1,prefix)
//	}
//end :
//	netWay.Option.Mylog.Warning("DemonSignal end")
//}
