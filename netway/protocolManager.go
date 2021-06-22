package netway

import (
	"context"
	"encoding/json"
	"errors"
	"frame_sync/myproto"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"regexp"
	"strconv"
	"strings"
	"zlib"
)

type ProtocolManager struct {
	TcpServer *TcpServer
	WsHttpServer      	*Httpd
	Option ProtocolManagerOption
	Close 	chan int
}

type ProtocolManagerOption struct {
	Ip 				string
	HttpPort 		string
	TcpPort 		string
	WsUri 			string
	OpenNewConnBack	func ( connFD FDAdapter)
}
//
func NewProtocolManager(option ProtocolManagerOption)*ProtocolManager{
	mylog.Info("NewProtocolManager instance:")
	protocolManager := new (ProtocolManager)
	protocolManager.Option = option
	protocolManager.Close = make(chan int)
	go protocolManager.l()
	return protocolManager
}
func  (protocolManager *ProtocolManager)Shutdown(){
	mylog.Alert("shutdown protocolManager")
	protocolManager.Close <- 1
}

func  (protocolManager *ProtocolManager)l(){
	<- protocolManager.Close
	protocolManager.TcpServer.Shutdown()
	protocolManager.WsHttpServer.shutdown()
}

func (protocolManager *ProtocolManager)Start(outCtx context.Context){
	//开始HTTP 监听 模块
	//go protocolManager.startHttpServer()
	//上面先注释掉了，WS的守护在HTTPD开启了
	httpdOption := HttpdOption {
		LogOption 	: myNetWay.Option.LogOption,
		RootPath 	: myNetWay.Option.HttpdRootPath,
		Ip			: myNetWay.Option.ListenIp,
		Port		: myNetWay.Option.WsPort,
		ParentCtx 	: myNetWay.Option.OutCxt,
		WsUri		: myNetWay.Option.WsUri,
		WsNewFDBack	: myProtocolManager.websocketHandler,
	}
	httpd := NewHttpd(httpdOption)
	protocolManager.WsHttpServer = httpd
	go httpd.startWs(myNetWay.Option.OutCxt)


	//tcp server
	myTcpServer :=  NewTcpServer(outCtx,protocolManager.Option.Ip,protocolManager.Option.TcpPort)
	protocolManager.TcpServer = myTcpServer
	go myTcpServer.Start()
}
func(protocolManager *ProtocolManager)websocketHandler( connFD *websocket.Conn) {
	mylog.Info("websocketHandler: have a new client")
	imp := WebsocketConnImpNew(connFD)
	protocolManager.Option.OpenNewConnBack(imp)
}

func(protocolManager *ProtocolManager)tcpHandler(tcpConn *TcpConn){
	imp := TcpConnImpNew(tcpConn)
	protocolManager.Option.OpenNewConnBack(imp)
}

func(protocolManager *ProtocolManager)udpHandler(){

}
//=======================================
//协议层的解包已经结束，这个时候需要将content内容进行转换成MSG结构
func  (protocolManager *ProtocolManager)parserContentMsg(msg myproto.Msg ,out interface{},playerId int32)error{
	content := msg.Content
	var err error
	//protocolCtrlInfo := myPlayerManager.GetPlayerCtrlInfoById(playerId)
	//contentType := protocolCtrlInfo.ContentType
	if msg.ContentType == CONTENT_TYPE_JSON {
		unTrunVarJsonContent := zlib.CamelToSnake([]byte(content))
		err = json.Unmarshal(unTrunVarJsonContent,out)
	}else if  msg.ContentType == CONTENT_TYPE_PROTOBUF {
		aaa := out.(proto.Message)
		err = proto.Unmarshal([]byte(content),aaa)
	}else{
		mylog.Error("parserContent err")
	}

	if err != nil{
		mylog.Error("parserMsgContent:",err.Error())
		return err
	}

	mylog.Debug("protocolManager parserMsgContent:",out)

	return nil
}
//解析C端发送的数据，这一层，对于用户层的content数据不做处理
//前2个字节控制流，3-6为协议号，7-38为sessionId
func  (protocolManager *ProtocolManager)parserContentProtocol(content string)(message myproto.Msg,err error){
	protocolSum := 6
	if len(content)<protocolSum{
		return message,errors.New("content < "+ strconv.Itoa(protocolSum))
	}
	if len(content)==protocolSum{
		errMsg := "content = "+strconv.Itoa(protocolSum)+" ,body is empty"
		return message,errors.New(errMsg)
	}
	ctrlStream := content[0:2]
	ctrlInfo := protocolManager.parserProtocolCtrlInfo([]byte(ctrlStream))
	actionIdStr := content[2:6]
	actionId,_ := strconv.Atoi(actionIdStr)
	actionName,empty := myProtocolActions.GetActionName(int32(actionId),"client")
	if empty{
		errMsg := "actionId ProtocolActions.GetActionName empty!!!"
		mylog.Error(errMsg,actionId)
		return message,errors.New("actionId ProtocolActions.GetActionName empty!!!")
	}

	//mylog.Info("parserContent actionid:",actionId, ",actionName:",actionName.Action)

	sessionId := ""
	userData := ""
	if actionName.Action != "login"{
		sessionId = content[6:38]
		userData = content[38:]
	}else{
		userData = content[6:]
	}

	msg := myproto.Msg{
		Action: actionName.Action,
		Content:userData,
		ContentType : ctrlInfo.ContentType,
		ProtocolType: ctrlInfo.ProtocolType,
		SessionId: sessionId,
	}
	//mylog.Debug("parserContentProtocol msg:",msg)
	return msg,nil
}


type ProtocolCtrlInfo struct {
	ContentType int32
	ProtocolType int32
}
func (protocolManager *ProtocolManager)parserProtocolCtrlInfo(stream []byte)ProtocolCtrlInfo{
	//firstByte := stream[0:1][0]
	//mylog.Debug("firstByte:",firstByte)
	//firstByteHighThreeBit := (firstByte >> 5 ) & 7
	//firstByteLowThreeBit := ((firstByte << 5 ) >> 5 )  & 7
	firstByteHighThreeBit , _:= strconv.Atoi(string(stream[0:1]))
	firstByteLowThreeBit , _:= strconv.Atoi(string(stream[1:2]))
	protocolCtrlInfo := ProtocolCtrlInfo{
		ContentType : int32(firstByteHighThreeBit),
		ProtocolType : int32(firstByteLowThreeBit),
	}
	//mylog.Debug("parserProtocolCtrlInfo ContentType:",protocolCtrlInfo.ContentType,",ProtocolType:",protocolCtrlInfo.ProtocolType)
	return protocolCtrlInfo
}

//将 结构体 压缩成 字符串
func  (protocolManager *ProtocolManager)CompressContent(contentStruct interface{},playerId int32)(content []byte  ,err error){
	protocolCtrlInfo := myPlayerManager.GetPlayerCtrlInfoById(playerId)
	contentType := protocolCtrlInfo.ContentType

	//mylog.Debug("CompressContent contentType:",contentType)
	if contentType == CONTENT_TYPE_JSON {
		//这里有个问题：纯JSON格式与PROTOBUF格式在PB文件上 不兼容
		//严格来说是GO语言与protobuf不兼容，即：PB文件的  结构体中的 JSON-TAG
		//PROTOBUF如果想使用驼峰式变量名，即：成员变量名区分出大小写，那必须得用<下划线>分隔，编译后，下划线转换成大写字母
		//编译完成后，虽然支持了驼峰变量名，但json-tag 并不是驼峰式，却是<下划线>式
		//那么，在不想改PB文件的前提下，就得在程序中做兼容

		//所以，先将content 字符串 由下划线转成 驼峰式
		content, err = json.Marshal(JsonCamelCase{contentStruct})
		//mylog.Info("CompressContent json:",string(content),err )
	}else if  contentType == CONTENT_TYPE_PROTOBUF {
		contentStruct := contentStruct.(proto.Message)
		content, err = proto.Marshal(contentStruct)
	}else{
		err = errors.New(" switch err")
	}
	if err != nil{
		mylog.Error("CompressContent err :",err.Error())
	}
	return content,err
}

type JsonCamelCase struct {
	Value interface{}
}
//下划线 转 驼峰命
func Case2Camel(name string) string {
	//将 下划线 转 空格
	name = strings.Replace(name, "_", " ", -1)
	//将 字符串的 每个 单词 的首字母转大写
	name = strings.Title(name)
	//最后再将空格删掉
	return strings.Replace(name, " ", "", -1)
}

func (c JsonCamelCase) MarshalJSON() ([]byte, error) {
	var keyMatchRegex = regexp.MustCompile(`\"(\w+)\":`)
	marshalled, err := json.Marshal(c.Value)
	converted := keyMatchRegex.ReplaceAllFunc(
		marshalled,
		func(match []byte) []byte {
			matchStr := string(match)
			key := matchStr[1 : len(matchStr)-2]
			resKey := zlib.Lcfirst(Case2Camel(key))
			return []byte(`"` + resKey + `":`)
		},
	)
	return converted, err
}