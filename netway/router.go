package netway

import (
	"frame_sync/myproto"
	"github.com/gorilla/websocket"
	"strconv"
	"zlib"
)

func(netWay *NetWay) Router(msg myproto.Msg,conn *Conn)(data interface{},err error){

	requestLogin := myproto.RequestLogin{}
	requestClientPong := myproto.RequestClientPong{}
	requestClientPing := myproto.RequestClientPing{}
	requestPlayerResumeGame := myproto.RequestPlayerResumeGame{}
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
			err = myProtocolManager.parserContentMsg(msg,&requestLogin,conn.PlayerId)
		case "clientPong"://
			err = myProtocolManager.parserContentMsg(msg,&requestClientPong,conn.PlayerId)
		case "playerResumeGame"://恢复未结束的游戏
			err = myProtocolManager.parserContentMsg(msg,&requestPlayerResumeGame,conn.PlayerId)
		case "playerOperations"://玩家推送操作指令
			err = myProtocolManager.parserContentMsg(msg,&requestPlayerOperations,conn.PlayerId)
		case "playerMatchSignCancel"://玩家取消报名等待
			err = myProtocolManager.parserContentMsg(msg,&requestPlayerMatchSignCancel,conn.PlayerId)
		case "gameOver"://游戏结束
			err = myProtocolManager.parserContentMsg(msg,&requestGameOver,conn.PlayerId)
		case "clientHeartbeat"://心跳
			err = myProtocolManager.parserContentMsg(msg,&requestClientHeartbeat,conn.PlayerId)
		case "playerMatchSign"://
			err = myProtocolManager.parserContentMsg(msg,&requestPlayerMatchSign,conn.PlayerId)
		case "playerReady"://玩家进入状态状态
			err = myProtocolManager.parserContentMsg(msg,&requestPlayerReady,conn.PlayerId)
		case "roomHistory"://一局副本的，所有历史操作记录
			err = myProtocolManager.parserContentMsg(msg,&requestRoomHistory,conn.PlayerId)
		case "getRoom"://
			err = myProtocolManager.parserContentMsg(msg,&requestGetRoom,conn.PlayerId)
		case "clientPing":
			err = myProtocolManager.parserContentMsg(msg,&requestClientPing,conn.PlayerId)
		case "playerOver":
			err = myProtocolManager.parserContentMsg(msg,&requestPlayerOver,conn.PlayerId)
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
			mySync.PlayerResumeGame(requestPlayerResumeGame,conn )
		case "playerOperations"://玩家推送操作指令
			mySync.ReceivePlayerOperation(requestPlayerOperations,conn,msg.Content)
		case "playerMatchSignCancel"://玩家取消报名等待
			myMatch.delOnePlayer(requestPlayerMatchSignCancel,conn)
		case "gameOver"://游戏结束
			mySync.GameOver(requestGameOver,conn)
		case "playerReady"://玩家进入状态状态
			mySync.PlayerReady(requestPlayerReady,conn)
		case "roomHistory"://一局副本的，所有历史操作记录
			mySync.RoomHistory(requestRoomHistory,conn)
		case "getRoom":
			mySync.GetRoom(requestGetRoom,conn)
		case "playerOver":
			mySync.PlayerOver(requestPlayerOver,conn)
		default:
			mylog.Error("Router err:",msg)
	}

	return data,nil
}
//发送一条消息给一个玩家，根据conn，同时将消息内容进行编码与压缩
//大部分通信都是这个方法
func(netWay *NetWay)SendMsgCompressByConn(conn *Conn,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByConn ", "" , " action:",action)
	contentByte ,_ := myProtocolManager.CompressContent(contentStruct,conn.PlayerId)
	netWay.SendMsg(conn,action,contentByte)
}
//发送一条消息给一个玩家，根据playerId，同时将消息内容进行编码与压缩
func(netWay *NetWay)SendMsgCompressByUid(playerId int32,action string , contentStruct interface{}){
	mylog.Info("SendMsgCompressByUid playerId:",playerId , " action:",action)
	contentByte ,_ := myProtocolManager.CompressContent(contentStruct,playerId)
	netWay.SendMsgByUid(playerId,action,contentByte)
}
//发送一条消息给一个玩家FD，
func(netWay *NetWay)SendMsgByUid(playerId int32,action string , content []byte){
	conn,ok := connManager.getConnPoolById(playerId)
	if !ok {
		mylog.Error("conn not in pool,maybe del.")
		return
	}
	netWay.SendMsg(conn,action,content)
}

func(netWay *NetWay)SendMsg(conn *Conn,action string,content []byte){
	//获取协议号结构体
	actionMapT,empty := myProtocolActions.GetActionId(action,"server")
	mylog.Info("SendMsg",actionMapT.Id,conn.PlayerId,action)
	if empty{
		mylog.Error("GetActionId empty",action)
		return
	}
	protocolCtrlInfo := myPlayerManager.GetPlayerCtrlInfoById(conn.PlayerId)
	contentType := protocolCtrlInfo.ContentType
	protocolType := protocolCtrlInfo.ProtocolType
	player ,_ := myPlayerManager.GetById(conn.PlayerId)
	SessionIdBtye := []byte(player.SessionId)
	content  = zlib.BytesCombine(SessionIdBtye,content)
	//协议号
	strId := strconv.Itoa(int(actionMapT.Id))
	//合并 协议号 + 消息内容体
	content = zlib.BytesCombine([]byte(strId),content)
	if conn.Status == CONN_STATUS_CLOSE {
		mylog.Error("Conn status =CONN_STATUS_CLOSE.")
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
		roomId := myPlayerManager.GetRoomIdByPlayerId(conn.PlayerId)
		roomSyncMetrics := RoomSyncMetricsPool[roomId]
		roomSyncMetrics.OutputNum++
		roomSyncMetrics.OutputSize = roomSyncMetrics.OutputSize + len(content)
	}
	mylog.Debug("final sendmsg ctrlInfo: contentType-",contentTypeStr," protocolType-",protocolTypeStr," pid-",strId)
	mylog.Debug("final sendmsg content:",content)
	if contentType == CONTENT_TYPE_PROTOBUF {
		conn.Write(content,websocket.BinaryMessage)
		//netWay.myWriteMessage(Conn,websocket.BinaryMessage,content)
	}else{
		conn.Write(content,websocket.TextMessage)
		//netWay.myWriteMessage(Conn,websocket.TextMessage,content)
	}
}
