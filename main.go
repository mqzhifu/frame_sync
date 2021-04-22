package main

import (
	"context"
	"os"
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
	mainChan := make(chan int )

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

