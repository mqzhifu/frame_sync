package netway

import (
	"encoding/json"
	"errors"
	"frame_sync/myproto"
	myprotocol "frame_sync/myprotocol"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"zlib"
)

type MyServer struct {
	Host 			string
	Port 			string
	MapSize 		int32
	RoomPeople 		int32
	Uri				string
	OffLineWaitTime int32
	ActionMap		map[string]map[int32]myprotocol.ActionMap
	ContentType		int32
	LoginAuthType	string
	FPS 			int32
}

//type ApiList struct {
//	ActionMap		map[string]map[int]myprotocol.ActionMap `json:"actionMap"`
	//JsonFormat 		map[int]string							`json:"jsonFormat"`
//}
type MyMetrics struct {
	Rooms	int `json:"room"`
	Players	int `json:"players"`
	Conns 	int `json:"conns"`
	InputNum int `json:"inputNum"`
	InputSize int `json:"inputSize"`
	OutputNum int `json:"outputNum"`
	OutputSize int `json:"outputSize"`
	InputErrNum int `json:"inputErrNum"`
}
var RoomList	map[string]Room
func uriTurnPath (uri string)string{
	n := strings.Index(uri,"?")
	if  n ==  - 1{
		return uri
	}
	uriByte := []byte(uri)
	path := uriByte[0:n]
	return string(path)
}
func  wwwHandler(w http.ResponseWriter, r *http.Request){
	//parameter := r.URL.Query()//GET 方式URL 中的参数 转 结构体
	uri := r.URL.RequestURI()
	mylog.Info("uri:",uri)
	if uri == "" || uri == "/" {
		ResponseStatusCode(w,500,"RequestURI is null or uir is :  '/'")
		return
	}
	//zlib.MyPrint(r.Header)
	uri = uriTurnPath(uri)
	query := r.URL.Query()
	var jsonStr []byte
	//var err error
	if uri == "/www/getServer"{
		options := mynetWay.Option
		//options.Host = "39.106.65.76"

		format := query.Get("format")
		format = strings.Trim(format," ")
		if format == ""{
			jsonStr,_ = json.Marshal(&options)
		}else if format == "proto"{
			cfgServer := myproto.CfgServer{
				ListenIp			:options.ListenIp,
				OutIp				:options.OutIp,
				WsPort				:options.WsPort,
				TcpPort				:options.TcpPort,
				UdpPort				:options.UdpPort,
				ContentType			:options.ContentType,
				LoginAuthType		:options.LoginAuthType,
				LoginAuthSecretKey	:options.LoginAuthSecretKey,
				IOTimeout			:options.IOTimeout,
				ConnTimeout			: options.ConnTimeout,
				Protocol			: options.Protocol,
				WsUri				: options.WsUri,
				MaxClientConnNum	:options.MaxClientConnNum,
				RoomPeople			:options.RoomPeople,
				RoomReadyTimeout 	:options.RoomReadyTimeout,
				OffLineWaitTime		:options.OffLineWaitTime,//玩家掉线后，等待多久
				MapSize				:options.MapSize,
				LockMode			: options.LockMode,
				FPS					:options.FPS,
				Store				: options.Store,
			}
			jsonStr,_ = proto.Marshal(&cfgServer)
		}

	}else if uri == "/www/apilist"{
		format := query.Get("format")
		info := mynetWay.ProtocolActions.GetActionMap()
		if format == ""{
			jsonStr,_ = json.Marshal(&info)
		}else if format == "proto"{
			cfgServer := myproto.CfgProtocolActions{}
			client := make(map[int32]*myproto.CfgActions)
			server := make(map[int32]*myproto.CfgActions)
			for k,v := range info["client"]{
				client[k] = &myproto.CfgActions{
					Id: v.Id,
					Action: v.Action,
					Desc: v.Desc,
					Demo: v.Demo,
				}
			}
			for k,v := range info["server"]{
				client[k] = &myproto.CfgActions{
					Id: v.Id,
					Action: v.Action,
					Desc: v.Desc,
					Demo: v.Demo,
				}
			}

			cfgServer.Client = client
			cfgServer.Server = server
			jsonStr,_ = proto.Marshal(&cfgServer)
		}

	}else if uri == "/www/getMetrics"{
		pool:= myMetrics.Pool
		pool["execTime"] = int(int(zlib.GetNowMillisecond()) - pool["starupTime"])
		jsonStr,_ = json.Marshal(&pool)
		zlib.MyPrint(string(jsonStr))
	}else if uri == "/www/getFD"{
	}else if uri == "/www/getRoomList"{
		type RoomList struct {
			Rooms map[string]Room              `json:"rooms"`
			Metrics map[string]RoomSyncMetrics `json:"metrics"`
		}

		myroomList := make(map[string]Room)
		roomListPoint := MySyncRoomPool
		myRoomMetrics := make(map[string]RoomSyncMetrics)
		//var emptyArr  []*ResponseRoomHistory
		if len(roomListPoint) > 0 {
			for k,v := range roomListPoint{
				tt := *v
				tt.LogicFrameHistory = nil
				myroomList[k] = tt
				myRoomMetrics[k] = RoomSyncMetricsPool[k]
			}
		}

		roomList := RoomList{
			Rooms: myroomList,
			Metrics: myRoomMetrics,
		}
		
		jsonStr,_ = json.Marshal(&roomList)
		//mylog.Debug("jsonStr:",jsonStr,err)
	} else if uri == "/www/actionMap"{

	}else if uri == "/www/getRoomOne"{
		roomId := query.Get("id")
		room := MySyncRoomPool[roomId]
		//history := room.LogicFrameHistory
		//type RoomOne struct {
		//	Info Room 	`json:"info"`
		//	HistoryList []*myproto.ResponseRoomHistory `json:"historyList"`
		//}
		//roonOne := RoomOne{}
		//roonOne.Info = *room
		//roonOne.HistoryList = history
		//zlib.MyPrint(roonOne)
		jsonStr,_ = json.Marshal(&room)
		//zlib.ExitPrint(jsonStr)
	} else if uri == "/www/createJwtToken"{
		randUid := query.Get("id")
		randUid = strings.Trim(randUid," ")
		if randUid == ""{
			jsonStr = []byte( "id 为空")
		}else{
			uidStrConvInt32,_ := strconv.ParseInt(randUid,10,32)
			payload := zlib.JwtDataPayload{
				Uid:int32(uidStrConvInt32),
				ATime:int32(zlib.GetNowMillisecond()),
				AppId:2,
			}
			token := zlib.CreateJwtToken(mynetWay.Option.LoginAuthSecretKey,payload)
			type CreateJwtNewToken struct {
				Uid 	int32
				Token 	string
			}
			createJwtNewToken := CreateJwtNewToken{
				Uid: payload.Uid,
				Token:token,
			}
			jsonStr,_ = json.Marshal(&createJwtNewToken)
		}

	} else if uri == "/www/testCreateJwtToken"{
		//info := mynetWay.testCreateJwtToken()
		//jsonStr,_ = json.Marshal(&info)
	}else if uri == "/www/getProtoFile"{
		filePath := "/myproto/api.proto"
		fileContent, err := getStaticFileContent(filePath)
		if err != nil{
			mylog.Error("/www/getProtoFile:",err.Error())
		}
		jsonStr = []byte(fileContent)
	}else{
		err := routeStatic(w,r,uri)
		if err != nil{
			return
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型

	w.Header().Set("Content-Length",strconv.Itoa( len(jsonStr) ) )
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(jsonStr)

}

func  getStaticFileContent(fileSuffix string)(content string ,err error){
	//code,msg = httpd.redisMetrics()
	baseDir ,_ := os.Getwd()
	//path := baseDir + "/../gamematch/www"
	path := baseDir
	mylog.Debug("final path:",path)
	filePath := path+fileSuffix
	mylog.Info("getStaticFileContent File path:",filePath)
	b, err := ioutil.ReadFile(filePath) // just pass the file name
	return string(b),err
}

func  routeStatic(w http.ResponseWriter,r *http.Request,uri string)error{
	//uriSplit := strings.Split(uri,"?")
	//if uriSplit[0] == "/apireq.html" {
	//	uri = uriSplit[0]
	//}
	if  uri == "/www/ws.html" ||
		uri == "/www/sync_frame_client_server.jpg" ||
		uri == "/www/jquery.min.js"||
		uri == "/www/sync.js"||
		uri == "/www/api_web_pb.js"||
		uri == "/www/roomlist.html"||
		uri == "/www/metrics.html"||
		uri == "/www/serverUpVersionMemo.html"||
		uri == "/www/sync_frame_client_server.jpg" ||
		uri == "/www/rsync_frame_lock_step.jpg" ||
		uri == "/www/index.html" ||
		uri == "/www/config.html" ||
		uri == "/www/roomDetail.html" ||
		uri == "/www/apilist.html"{ //静态文件

		fileContent, err := getStaticFileContent(uri)
		if err != nil {
			ResponseStatusCode(w, 404, err.Error())
			return errors.New("routeStatic 404")
		}
		//踦域处理
		w.Header().Set("Access-Control-Allow-Origin","*")
		w.Header().Add("Access-Control-Allow-Headers","Content-Type")
		//w.Header().Set("content-type","text/plain")

		w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
		w.Write([]byte(fileContent))
	}
	return nil
}

//http 响应状态码
func  ResponseStatusCode(w http.ResponseWriter,code int ,responseInfo string){
	mylog.Info("ResponseStatusCode",code,responseInfo)

	w.Header().Set("Content-Length",strconv.Itoa( len(responseInfo) ) )
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(403)
	w.Write([]byte(responseInfo))
}