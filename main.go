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

	//type Ccc struct {
	//
	//}
	//
	//ccc:= Ccc{}
	//type Aaa struct {
	//	Bbb interface{}
	//}
	//
	//aaa := Aaa{
	//
	//	Bbb: ccc,
	//}
	//
	//zlib.ExitPrint(aaa.Bbb.(Ccc))

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
		//Host 				:"192.168.192.125",
		//Host 				:"192.168.192.91",
		//Host 				:"192.168.192.132",
		Host 				:"192.168.192.97",
		//Host 				:"127.0.0.1",
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
		RoomPeople			:4,
		MapSize				:10,
		OffLineWaitTime		:20,
	}
	newNetWay := NewNetWay(newNetWayOption)
	go newNetWay.Startup()


	for{
		select {
		case   <-mainChan:
			mylog.Warning("mainChan")
			goto eee
		default:
			time.Sleep(time.Second * 1)
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

