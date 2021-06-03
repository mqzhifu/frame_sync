package netway

import (
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

func(netWay *NetWay) Router(msg myproto.Msg,conn *Conn)(data interface{},err error){

	requestLogin := myproto.RequestLogin{}
	requestClientPong := myproto.RequestClientPong{}
	requestClientPing := myproto.RequestClientPing{}
	requestPlayerResumeGame := myproto.RequestPlayerResumeGame{}
	//requestPlayerStatus := RequestPlayerStatus{}
	requestPlayerOperations := myproto.RequestPlayerOperations{}
	requestPlayerMatchSign := myproto.RequestPlayerMatchSign{}
	requestPlayerMatchSignCancel := myproto.RequestPlayerMatchSignCancel{}
	requestGameOver := myproto.RequestGameOver{}
	requestClientHeartbeat := myproto.RequestClientHeartbeat{}
	requestPlayerReady := myproto.RequestPlayerReady{}
	requestRoomHistory := myproto.RequestRoomHistory{}
	requestGetRoom := myproto.RequestGetRoom{}
	requestPlayerOver := myproto.RequestPlayerOver{}

	//这里有个BUG，LOGIN 函数只能在第一次调用，回头加个限定
	switch msg.Action {
		case "login"://
			err = netWay.parserContentMsg(msg,&requestLogin,conn.PlayerId)
		case "clientPong"://
			err = netWay.parserContentMsg(msg,&requestClientPong,conn.PlayerId)
		case "playerResumeGame"://恢复未结束的游戏
			err = netWay.parserContentMsg(msg,&requestPlayerResumeGame,conn.PlayerId)
		//case "playerStatus"://玩家检测是否有未结束的游戏
		//	err = parserMsgContent(msg.Content,&requestPlayerStatus)
		//netWay.Players.getPlayerStatus(requestPlayerStatus,wsConn)
		case "playerOperations"://玩家推送操作指令
			err = netWay.parserContentMsg(msg,&requestPlayerOperations,conn.PlayerId)
		case "playerLogicFrameAck":
			//err = parserMsgContent(msg.Content,&logicFrame)
			//mySync.playerLogicFrameAck(logicFrame,wsConn)
		case "playerMatchSignCancel"://玩家取消报名等待
			err = netWay.parserContentMsg(msg,&requestPlayerMatchSignCancel,conn.PlayerId)
		case "gameOver"://游戏结束
			err = netWay.parserContentMsg(msg,&requestGameOver,conn.PlayerId)
		case "clientHeartbeat"://心跳
			err = netWay.parserContentMsg(msg,&requestClientHeartbeat,conn.PlayerId)
		case "playerMatchSign"://
			err = netWay.parserContentMsg(msg,&requestPlayerMatchSign,conn.PlayerId)
		case "playerReady"://玩家进入状态状态
			err = netWay.parserContentMsg(msg,&requestPlayerReady,conn.PlayerId)
		case "roomHistory"://一局副本的，所有历史操作记录
			err = netWay.parserContentMsg(msg,&requestRoomHistory,conn.PlayerId)
		case "getRoom"://
			err = netWay.parserContentMsg(msg,&requestGetRoom,conn.PlayerId)
		case "clientPing":
			err = netWay.parserContentMsg(msg,&requestClientPing,conn.PlayerId)
		case "playerOver":
			err = netWay.parserContentMsg(msg,&requestPlayerOver,conn.PlayerId)
		default:
			mylog.Error("Router err:",msg)
			return data,nil
	}
	if err != nil{
		return data,err
	}
	mylog.Info("Router ",msg.Action)
	switch msg.Action {
		case "login"://
			jwtData ,err := netWay.login(requestLogin,conn)
			return jwtData,err
		case "clientPong"://
			netWay.ClientPong(requestClientPong,conn)
		case "clientHeartbeat"://心跳
			netWay.heartbeat(requestClientHeartbeat,conn)
		case "playerMatchSign"://
			myMatch.addOnePlayer(requestPlayerMatchSign,conn)
		case "clientPing"://
			netWay.clientPing(requestClientPing,conn)
		case "playerResumeGame"://恢复未结束的游戏
			netWay.mySync.PlayerResumeGame(requestPlayerResumeGame,conn )
		case "playerOperations"://玩家推送操作指令
			netWay.mySync.ReceivePlayerOperation(requestPlayerOperations,conn,msg.Content)
		case "playerMatchSignCancel"://玩家取消报名等待
			myMatch.delOnePlayer(requestPlayerMatchSignCancel,conn)
		case "gameOver"://游戏结束
			netWay.mySync.GameOver(requestGameOver,conn)
		case "playerReady"://玩家进入状态状态
			netWay.mySync.PlayerReady(requestPlayerReady,conn)
		case "roomHistory"://一局副本的，所有历史操作记录
			netWay.mySync.RoomHistory(requestRoomHistory,conn)
		case "getRoom":
			netWay.mySync.GetRoom(requestGetRoom,conn)
		case "playerOver":
			netWay.mySync.PlayerOver(requestPlayerOver,conn)
		default:
			mylog.Error("Router err:",msg)

		//case "playerAddRoom"://玩家进入房间
	}

	return data,nil
}
//发送一条消息给一个玩家FD，同时将消息内容进行编码与压缩
func(netWay *NetWay)SendMsgCompressByConn(conn *Conn,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByConn ", "" , " action:",action)
	contentByte ,_ := netWay.CompressContent(contentStruct,conn.PlayerId)
	netWay.SendMsg(conn,action,contentByte)
}
//发送一条消息给一个玩家FD，同时将消息内容进行编码与压缩
func(netWay *NetWay)SendMsgCompressByUid(playerId int32,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByUid playerId:",playerId , " action:",action)
	contentByte ,_ := netWay.CompressContent(contentStruct,playerId)
	netWay.SendMsgByUid(playerId,action,contentByte)
}
func(netWay *NetWay)SendMsg(conn *Conn,action string,content []byte){
	//获取协议号结构体
	actionMapT,empty := netWay.ProtocolActions.GetActionId(action,"server")
	mylog.Info("SendMsg",actionMapT.Id,conn.PlayerId,action)
	if empty{
		mylog.Error("GetActionId empty",action)
		return
	}
	protocolCtrlInfo := netWay.PlayerManager.GetPlayerCtrlInfoById(conn.PlayerId)
	contentType := protocolCtrlInfo.ContentType
	protocolType := protocolCtrlInfo.ProtocolType
	player ,_ := netWay.PlayerManager.GetById(conn.PlayerId)
	SessionIdBtye := []byte(player.SessionId)
	content  = zlib.BytesCombine(SessionIdBtye,content)
	//协议号
	strId := strconv.Itoa(int(actionMapT.Id))
	//合并 协议号 + 消息内容体
	content = zlib.BytesCombine([]byte(strId),content)
	if conn.Status == CONN_STATUS_CLOSE {
		mylog.Error("wsConn status =CONN_STATUS_CLOSE.")
		return
	}

	//var protocolCtrlFirstByteArr []byte
	//contentTypeByte := byte(contentType)
	//protocolTypeByte := byte(player.ProtocolType)
	//contentTypeByteRight := contentTypeByte >> 5
	//protocolCtrlFirstByte := contentTypeByteRight | protocolTypeByte
	//protocolCtrlFirstByteArr = append(protocolCtrlFirstByteArr,protocolCtrlFirstByte)
	//content = zlib.BytesCombine(protocolCtrlFirstByteArr,content)
	contentTypeStr := strconv.Itoa(int(contentType))
	protocolTypeStr := strconv.Itoa(int(protocolType))
	contentTypeAndprotocolType := contentTypeStr + protocolTypeStr
	content = zlib.BytesCombine([]byte(contentTypeAndprotocolType),content)
	//myMetrics.IncNode("output_num")
	//myMetrics.PlusNode("output_size",len(content))
	//房间做统计处理
	if action =="pushLogicFrame"{
		roomId := netWay.PlayerManager.GetRoomIdByPlayerId(conn.PlayerId)
		roomSyncMetrics := RoomSyncMetricsPool[roomId]
		roomSyncMetrics.OutputNum++
		roomSyncMetrics.OutputSize = roomSyncMetrics.OutputSize + len(content)
	}
	mylog.Debug("final sendmsg ctrlInfo: contentType-",contentTypeStr," protocolType-",protocolTypeStr," pid-",strId)
	mylog.Debug("final sendmsg content:",content)
	if contentType == CONTENT_TYPE_PROTOBUF {
		conn.Write(content,websocket.BinaryMessage)
		//netWay.myWriteMessage(wsConn,websocket.BinaryMessage,content)
	}else{
		conn.Write(content,websocket.TextMessage)
		//netWay.myWriteMessage(wsConn,websocket.TextMessage,content)
	}
}
//发送一条消息给一个玩家FD，
func(netWay *NetWay)SendMsgByUid(playerId int32,action string , content []byte){
	wsConn,ok := connManager.getConnPoolById(playerId)
	if !ok {
		mylog.Error("wsConn not in pool,maybe del.")
		return
	}
	netWay.SendMsg(wsConn,action,content)
}
//解析C端发送的数据，这一层，对于用户层的content数据不做处理
//前2个字节控制流，3-6为协议号，7-38为sessionId
func  (netWay *NetWay)parserContentProtocol(content string)(message myproto.Msg,err error){
	protocolSum := 6
	if len(content)<protocolSum{
		return message,errors.New("content < "+ strconv.Itoa(protocolSum))
	}
	if len(content)==protocolSum{
		errMsg := "content = "+strconv.Itoa(protocolSum)+" ,body is empty"
		return message,errors.New(errMsg)
	}
	ctrlStream := content[0:2]
	ctrlInfo := netWay.parserProtocolCtrlInfo([]byte(ctrlStream))
	actionIdStr := content[2:6]
	actionId,_ := strconv.Atoi(actionIdStr)
	actionName,empty := netWay.ProtocolActions.GetActionName(int32(actionId),"client")
	if empty{
		errMsg := "actionId ProtocolActions.GetActionName empty!!!"
		mylog.Error(errMsg,actionId)
		return message,errors.New("actionId ProtocolActions.GetActionName empty!!!")
	}

	mylog.Info("parserContent actionid:",actionId, ",actionName:",actionName.Action)

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
	mylog.Debug("msg:",msg)
	return msg,nil
}
type ProtocolCtrlInfo struct {
	ContentType int32
	ProtocolType int32
}
func (netWay *NetWay)parserProtocolCtrlInfo(stream []byte)ProtocolCtrlInfo{
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
	mylog.Info("parserProtocolCtrlInfo ContentType:",protocolCtrlInfo.ContentType,",ProtocolType:",protocolCtrlInfo.ProtocolType)
	return protocolCtrlInfo
}
//协议层的解包已经结束，这个时候需要将content内容进行转换成MSG结构
func (netWay *NetWay)parserContentMsg(msg myproto.Msg ,out interface{},playerId int32)error{
	content := msg.Content
	var err error
	//protocolCtrlInfo := netWay.PlayerManager.GetPlayerCtrlInfoById(playerId)
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

	mylog.Debug("netWay parserMsgContent:",out)

	return nil
}
//将 结构体 压缩成 字符串
func (netWay *NetWay)CompressContent(contentStruct interface{},playerId int32)(content []byte  ,err error){
	protocolCtrlInfo := netWay.PlayerManager.GetPlayerCtrlInfoById(playerId)
	contentType := protocolCtrlInfo.ContentType

	mylog.Debug("CompressContent contentType:",contentType)
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
