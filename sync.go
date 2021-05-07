package main

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"zlib"
)

type Sync struct {

}
//type LogicFrameHistory struct {
//	Id 		int
//	Action 	string
//	Content string
//}
//逻辑帧
type LogicFrame struct {
	Id 				int 		`json:"id"`
	RoomId 			string		`json:"roomId"`
	SequenceNumber 	int			`json:"sequenceNumber"`
	Operations 		[]Operation	`json:"operations"`
}
//玩家操作指令集
//type Operation struct {
//	Id 			int 	`json:"id"`
//	Event 		string	`json:"event"`
//	Value 		string	`json:"value"`
//	PlayerId 	int32		`json:"playerId"`
//}
var testLock int
//集合 ：每个副本
var mySyncRoomPool map[string]*Room
//索引表，PlayerId=>RoomId
var mySyncPlayerRoom map[int32]string
var logicFrameMsgDefaultId int32
var operationDefaultId int32
func NewSync()*Sync{
	mylog.Info("NewSync instance")
	mySyncRoomPool = make(map[string]*Room)
	mySyncPlayerRoom = make(map[int32]string)
	sync := new(Sync)

	logicFrameMsgDefaultId = 16
	operationDefaultId = 32

	testLock = 0

	return sync
}
func (sync *Sync)checkRoomTimeoutLoop(ctx context.Context){
	for{
		select {
		case <- ctx.Done():
			mylog.Warning("checkRoomTimeoutLoop done.")
			return
		default:
			sync.checkRoomTimeout()
			time.Sleep(1 * time.Second)
		}
	}
}
func (sync *Sync)checkRoomTimeout(){
	roomLen := len(mySyncRoomPool)
	if roomLen <= 0{
		return
	}

	now := int32( zlib.GetNowTimeSecondToInt())
	for _,v:=range mySyncRoomPool{
		if v.Status != ROOM_STATUS_EXECING{
			continue
		}

		if now > v.Timeout{
			sync.roomEnd(v.Id)
		}
	}
}

//给集合添加一个新的 游戏副本
func (sync *Sync) addPoolElement(room 	*Room){
	mynetWay.Option.Mylog.Info("addPoolElement")
	_ ,exist := mySyncRoomPool[room.Id]
	if exist{
		mynetWay.Option.Mylog.Error("mySyncRoomPool has exist : ",room.Id)
		return
	}
	//syncRoomPoolElement := SyncRoomPoolElement{
	//	Status: SYNC_ELEMENT_STATUS_WAIT,
	//	Room: room,
	//	SequenceNumber: 0,
	//	PlayersAckList: make(map[int]int),
	//	PlayersAckStatus:PLAYERS_ACK_STATUS_INIT,
	//	AddTime: zlib.GetNowTimeSecondToInt(),
	//	RandSeek: zlib.GetRandIntNum(100),
	//
	//}
	mySyncRoomPool[room.Id] = room
}
//进入战后，场景渲染完后，进入准确状态
func (sync *Sync)playerReady(requestPlayerReady RequestPlayerReady ,wsConn *WsConn) {
	roomId := mySyncPlayerRoom[requestPlayerReady.PlayerId]
	mylog.Debug("mySyncPlayerRoom:",mySyncPlayerRoom)
	room,empty := mySync.getPoolElementById(roomId)
	if empty{
		mylog.Error("playerReady getPoolElementById empty",roomId)
		return
	}
	room.PlayersReadyList[requestPlayerReady.PlayerId] = 1
	playerReadyCnt := 0
	mylog.Info("room.PlayersReadyList:",room.PlayersReadyList)
	for _,v := range room.PlayersReadyList{
		if v == 1{
			playerReadyCnt++
		}
	}

	if playerReadyCnt < len(room.PlayersReadyList)  {
		mylog.Error("now ready cnt :",playerReadyCnt," ,so please wait other players...")
		return
	}
	responseStartBattle := ResponseStartBattle{
		SequenceNumberStart: int32(0),
	}
	//jsonSTR,_ := json.Marshal(responseStartBattle)
	//mySync.boardCastInRoom(room.Id,"startBattle",string(jsonSTR))
	mySync.boardCastInRoom(room.Id,"startBattle",&responseStartBattle)
	room.upStatus(ROOM_STATUS_EXECING)
	//初始结束后，这里方便测试，再补一帧，所有玩家的随机位置
	if room.PlayerList[0].Id < 999{
		var operations []*Operation
		for _,player:= range room.PlayerList{
			location := strconv.Itoa(zlib.GetRandInt32Num(mynetWay.Option.MapSize)) + "," + strconv.Itoa(zlib.GetRandInt32Num(mynetWay.Option.MapSize))
			operation := Operation{
				Id: logicFrameMsgDefaultId,
				Event: "move",
				Value: location,
				PlayerId: player.Id,
			}
			operations = append(operations,&operation)
		}
		logicFrameMsg := ResponsePushLogicFrame{
			Id	: operationDefaultId,
			RoomId: room.Id,
			SequenceNumber :int32(room.SequenceNumber),
			Operations 		:operations,
		}
		//logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
		//mySync.boardCastInRoom(room.Id,"pushLogicFrame",string(logicFrameMsgJson))
		mySync.boardCastInRoom(room.Id,"pushLogicFrame",&logicFrameMsg)
	}

	go sync.logicFrameLoop(room)

}

//一个新的房间，开始同步
func  (sync *Sync)start(roomId string){
	mynetWay.Option.Mylog.Warning("start a new game:")

	room,_ := sync.getPoolElementById(roomId)
	mynetWay.Option.Mylog.Warning("roomInfo:",room)
	room.upStatus(ROOM_STATUS_READY)
	responseClientInitRoomData := ResponseEnterBattle{
		Status 			:room.Status,
		AddTime 		:room.AddTime,
		RoomId			:roomId,
		SequenceNumber	: -1,
		PlayerList		:room.PlayerList,
		RandSeek		:room.RandSeek,
		Time			:time.Now().UnixNano() / 1e6,
	}

	for _,v :=range room.PlayerList{
		mySyncPlayerRoom[v.Id] = roomId
	}

	//zlib.MyPrint("old:",responseClientInitRoomData)
	//jsonStrByte,_ := json.Marshal(responseClientInitRoomData)
	//sync.boardCastInRoom(roomId,"enterBattle",string(jsonStrByte))
	sync.boardCastInRoom(roomId,"enterBattle",&responseClientInitRoomData)
	room.CloseChan = make(chan int)
}
//有一个房间内，搜索一个用户
func (sync *Sync)getPlayerByIdInRoom(playerId int32 ,room *Room,)(myplayer *Player,empty bool){
	for _,player:= range room.PlayerList{
		if player.Id == playerId{
			return player,false
		}
	}
	return myplayer,true
}

func (sync *Sync) getPoolElementById(roomId string)(SyncRoomPoolElement *Room,empty bool){
	v,exist := mySyncRoomPool[roomId]
	if !exist{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty,",roomId)
		return SyncRoomPoolElement,true
	}
	return v,false
}

func  (sync *Sync)logicFrameLoop(room *Room){
	if mynetWay.Option.FPS > 1000 {
		zlib.ExitPrint("fps > 1000 ms")
	}

	fpsTime :=  1000 /  mynetWay.Option.FPS
	i := 0
	for{
		select {
		case   <-room.CloseChan:
			goto end
		default:
			sleepMsTime := sync.logicFrameLoopReal(room,fpsTime)
			sleepMsTimeD := time.Duration(sleepMsTime)
			if sleepMsTime > 0 {
				time.Sleep(sleepMsTimeD * time.Millisecond)
			}
			i++
			//if i > 10{
			//	zlib.ExitPrint(1111)
			//}
		}
	}
end:
	mylog.Warning("pushLogicFrame loop routine close")
}

func  (sync *Sync)logicFrameLoopReal(room *Room,fpsTime int32)int32{
	queue := room.PlayersOperationQueue
	end := queue.Len()
	if end <= 0 {
		return fpsTime
	}

	if mynetWay.Option.LockMode == LOCK_MODE_PESSIMISTIC{
		ack := 0
		for _, v := range room.PlayersAckList {
			if v == 1 {
				ack++
			}
		}

		if ack < len(room.PlayersAckList) {
			mylog.Error("还有玩家未发送操作记录,当前确认人数:",ack)
			return fpsTime
		}
		sync.upSyncRoomPoolElementPlayersAckStatus(room.Id, PLAYERS_ACK_STATUS_OK)
	}


	room.SequenceNumber++

	logicFrame := ResponsePushLogicFrame{
		Id:             0,
		RoomId:         room.Id,
		SequenceNumber: int32(room.SequenceNumber),
	}
	var operations []*Operation
	i := 0
	element := queue.Front()

	for {
		if i >= end {
			break
		}
		operationsValueInterface := element.Value
		operationsValue := operationsValueInterface.(string)
		var elementOperations []Operation
		err := json.Unmarshal([]byte(operationsValue), &elementOperations)
		if err != nil {
			mylog.Error("queue json.Unmarshal err :", err.Error())
		}
		//mylog.Debug(operationsValue,"elementOperations",elementOperations)
		for j := 0; j < len(elementOperations); j++ {
			//if elementOperations[j].Event != "empty"{
			//	mylog.Debug("elementOperations j :",elementOperations[j])
			//	testLock = 1
			//}
			operations = append(operations, &elementOperations[j])
		}



		tmpElement := element.Next()
		queue.Remove(element)
		element = tmpElement

		i++
	}

	//if testLock == 1{
	//	zlib.MyPrint(logicFrame)
	//}
	mylog.Info("operations:",operations)
	logicFrame.Operations = operations
	//zlib.ExitPrint(logicFrame)
	//logicFrameStr, _ := json.Marshal(logicFrame)
	//sync.boardCastFrameInRoom(room.Id, "pushLogicFrame", string(logicFrameStr))
	sync.boardCastFrameInRoom(room.Id, "pushLogicFrame",&logicFrame)
	return fpsTime
}

func  (sync *Sync)receivePlayerOperation(logicFrame RequestPlayerOperations,wsConn *WsConn){
	//mylog.Debug(logicFrame)
	room,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",logicFrame.RoomId)
	}
	err := sync.checkReceiveOperation(room,logicFrame,wsConn)
	if err != nil{
		mylog.Error("receivePlayerOperation check error:",err.Error())
		return
	}
	if len(logicFrame.Operations) <= 0{
		mylog.Error("len(logicFrame.Operations) <= 0")
		return
	}
	//zlib.ExitPrint(logicFrame)
	//room.PlayersAckList[wsConn.PlayerId] = 1
	logicFrameStr ,_ := json.Marshal(logicFrame.Operations)
	room.PlayersOperationQueue.PushBack(string(logicFrameStr))
	if room.PlayersAckList[wsConn.PlayerId] == 0{
		room.PlayersAckList[wsConn.PlayerId] = 1
	}
}
func  (sync *Sync)checkReceiveOperation(room *Room,logicFrame RequestPlayerOperations,wsConn *WsConn)error{
	if room.Status != ROOM_STATUS_EXECING{
		return errors.New("room status err NOT execing, this status : "+strconv.Itoa(int(room.Status)))
	}
	numberMsg := "cli_sn:" +strconv.Itoa(int(logicFrame.SequenceNumber)) + ", now_sn:" + strconv.Itoa(room.SequenceNumber)
	if int(logicFrame.SequenceNumber) == room.SequenceNumber{
		mylog.Info("checkReceiveOperation ok , "+numberMsg)
		return nil
	}


	if int(logicFrame.SequenceNumber) > room.SequenceNumber{
		return errors.New("client num > room.SequenceNumber err:"+numberMsg)
	}
	//客户端延迟较高 相对的  服务端 发送较快
	if int(logicFrame.SequenceNumber) < room.SequenceNumber{
		return errors.New("client num < room.SequenceNumber err:"+numberMsg)
	}

	return nil
}

//func (sync *Sync) playerLogicFrameAck(logicFrame LogicFrame,wsConn *WsConn ){
//	mynetWay.Option.Mylog.Info("playerLogicFrameAck",wsConn.PlayerId)
//
//	syncRoomPoolElement,empty := sync.getPoolElementById(logicFrame.RoomId)
//	if empty{
//		mynetWay.Option.Mylog.Error("roomId is wrong",logicFrame.RoomId)
//		return
//	}
//	if syncRoomPoolElement.SequenceNumber != logicFrame.SequenceNumber{
//		mynetWay.Option.Mylog.Error("SequenceNumber wrong",logicFrame.SequenceNumber,syncRoomPoolElement.SequenceNumber)
//		return
//	}
//	if syncRoomPoolElement.PlayersAckStatus != PLAYERS_ACK_STATUS_WAIT{
//		mynetWay.Option.Mylog.Error("该帧号的状态!=待确认 状态,当前状态为：",syncRoomPoolElement.PlayersAckStatus)
//		return
//	}
//
//	player ,empty := sync.getPlayerByIdInRoom(wsConn.PlayerId,syncRoomPoolElement)
//	if empty{
//		mynetWay.Option.Mylog.Error("playerId wrong,not found in this room.",wsConn.PlayerId)
//		return
//	}
//	if syncRoomPoolElement.PlayersAckList[player.Id] == 1{
//		mynetWay.Option.Mylog.Error("该玩家已经确认过了，不要重复操作")
//		return
//	}else{
//		syncRoomPoolElement.PlayersAckList[player.Id] = 1
//	}
//
//	ack := 0
//	for _,v := range syncRoomPoolElement.PlayersAckList{
//		if v == 1{
//			ack++
//		}
//	}
//	mynetWay.Option.Mylog.Info("this room LogicFrameSequenceNumber(",syncRoomPoolElement.SequenceNumber,") ack total:",ack)
//
//	//该逻辑帧已全部确认
//	if ack == len(syncRoomPoolElement.PlayersAckList){
//		mynetWay.Option.Mylog.Info("one logic frame all ack.")
//		//更新 帧号 变更状态
//		mylog.Info("syncRoomPoolElement.SequenceNumber inc: old",syncRoomPoolElement.SequenceNumber)
//		syncRoomPoolElement.SequenceNumber++
//		//mylog.Info("syncRoomPoolElement.SequenceNumber inc: new",syncRoomPoolElement.SequenceNumber)
//		syncRoomPoolElement.PlayersAckList = make(map[int32]int32)
//		sync.upSyncRoomPoolElementPlayersAckStatus(logicFrame.RoomId,PLAYERS_ACK_STATUS_INIT)
//		//syncRoomPoolElement.PlayersAckStatus = PLAYERS_ACK_STATUS_INIT
//		mylog.Notice("have a new SequenceNumber:",syncRoomPoolElement.SequenceNumber)
//		//这里有个特殊处理，首帧其实是初始化的一些数据，当完成后，第一帧的动作按说应该是玩家操作
//		//但这里，先模拟一下，将玩家随机散落到一地图上的一些点
//
//
//		if syncRoomPoolElement.SequenceNumber == 1 {
//			mynetWay.Option.Mylog.Info("syncRoomPoolElement.SequenceNumber == 0")
//			//S端，每一逻辑帖，建立一个集合，保存广播的消息，玩家返回的确认ACK
//			var operations []Operation
//			for _,player:= range syncRoomPoolElement.PlayerList{
//				location := strconv.Itoa(zlib.GetRandInt32Num(mynetWay.Option.MapSize)) + "," + strconv.Itoa(zlib.GetRandInt32Num(mynetWay.Option.MapSize))
//				operation := Operation{
//					Id: logicFrameMsgDefaultId,
//					Event: "move",
//					Value: location,
//					PlayerId: player.Id,
//				}
//				operations = append(operations,operation)
//			}
//			logicFrameMsg := LogicFrame{
//				Id	: operationDefaultId,
//				RoomId: logicFrame.RoomId,
//				SequenceNumber :syncRoomPoolElement.SequenceNumber,
//				Operations 		:operations,
//			}
//			//logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
//			//sync.boardCastFrameInRoom(logicFrame.RoomId,"pushLogicFrame",string(logicFrameMsgJson))
//			sync.boardCastFrameInRoom(logicFrame.RoomId,"pushLogicFrame",logicFrameMsg)
//		}
//	}
//
//	mynetWay.Option.Mylog.Info("playerLogicFrameAck~ finish.")
//	return
//
//}
func  (sync *Sync)roomEnd(roomId string){
	mylog.Info("roomEnd")
	room,empty := sync.getPoolElementById(roomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",roomId)
		return
	}
	room.upStatus(ROOM_STATUS_END)
	for _,v:= range room.PlayerList{
		mynetWay.Players.upPlayerRoomId(v.Id,"")
		delete(mySyncPlayerRoom,v.Id)
	}
	room.CloseChan <- 1
}
//玩家操作后，触发C端主动发送游戏结束事件
func  (sync *Sync)gameOver(requestGameOver RequestGameOver,wsConn *WsConn){
	responseGameOver := ResponseGameOver{
		PlayerId : requestGameOver.PlayerId,
		RoomId:requestGameOver.RoomId,
		SequenceNumber: requestGameOver.SequenceNumber,
		Result:requestGameOver.Result,
	}
	sync.boardCastInRoom(requestGameOver.RoomId,"gameOver",&responseGameOver)

	sync.roomEnd(requestGameOver.RoomId)
	//jsonStr ,_ := json.Marshal(requestGameOver)
}
func  (sync *Sync)playerOver(requestGameOver RequestPlayerOver ,wsConn *WsConn){
	roomId := mySyncPlayerRoom[requestGameOver.PlayerId]
	responseOtherPlayerOver := ResponseOtherPlayerOver{PlayerId: requestGameOver.PlayerId}
	sync.boardCastInRoom(roomId,"otherPlayerOver",&responseOtherPlayerOver)
}

func (sync *Sync)upSyncRoomPoolElementPlayersAckStatus(roomId string,status int){
	syncRoomPoolElement,_  := sync.getPoolElementById(roomId)
	mylog.Notice("upSyncRoomPoolElementPlayersAckStatus ,old :",syncRoomPoolElement.PlayersAckStatus,"new",status)
	syncRoomPoolElement.PlayersAckStatus = status
}
//一个玩家取消了准备/报名
func (sync *Sync)cancelSign(requestCancelSign RequestPlayerMatchSignCancel,wsConn *WsConn){
	myMatch.delOnePlayer(wsConn.PlayerId)
}
//判定一个房间内，玩家在线的人
func (sync *Sync)roomOnlinePlayers(room *Room)[]int32{
	var playerOnLine []int32
	for _, v := range room.PlayerList {
		player, empty := mynetWay.Players.getById(v.Id)
		if empty {
			continue
		}
		if player.Status == PLAYER_STATUS_ONLINE {
			playerOnLine = append(playerOnLine,player.Id)
		}
	}
	return playerOnLine
}
//玩家断开连接后
func (sync *Sync)close(wsConn *WsConn){
	mylog.Warning("sync.close")
	//获取该玩家的roomId
	roomId,ok := mySyncPlayerRoom[wsConn.PlayerId]
	if !ok || roomId == "" {
		return
	}
	//判断下所有玩家是否均下线了
	room, _ := sync.getPoolElementById(roomId)
	if room.Status == ROOM_STATUS_EXECING{
		playerOnLine := sync.roomOnlinePlayers(room)
		playerOnLineCount := len(playerOnLine)
		playerOnLineCount-- //这里因为，已有一个玩家关闭中，但是还未处理
		mylog.Info("ROOM_STATUS_EXECING ,has check roomEnd , playerOnLineCount : ", playerOnLineCount)
		if playerOnLineCount <= 0 {
			sync.roomEnd(roomId)
		}else{
			responseOtherPlayerOffline := ResponseOtherPlayerOffline{
				PlayerId: wsConn.PlayerId,
			}
			//jsonStr,_ := json.Marshal(responseOtherPlayerOffline)
			sync.boardCastInRoom(roomId,"otherPlayerOffline",&responseOtherPlayerOffline)
		}
	}
}
//单纯的给一个房间里的人发消息，不考虑是否有顺序号的情况
func  (sync *Sync)boardCastInRoom(roomId string,action string ,contentStruct interface{}){
	room,empty  := sync.getPoolElementById(roomId)
	if empty {
		zlib.ExitPrint("syncRoomPoolElement is empty!!!")
	}
	for _,player:= range room.PlayerList {
		if player.Status == PLAYER_STATUS_OFFLINE{
			mylog.Error("player offline")
			continue
		}
		//mynetWay.SendMsgByUid(player.Id,action,content)
		mynetWay.SendMsgCompressByUid(player.Id,action,contentStruct)
	}
	content ,_:= json.Marshal(contentStruct)
	sync.addOneRoomHistory(room,action,string(content))
}
//给一个副本里的所有玩家广播数据，且该数据必须得有C端ACK
func  (sync *Sync)boardCastFrameInRoom(roomId string,action string ,contentStruct interface{}){
	mylog.Notice("boardCastFrameInRoom:",roomId,action)
	syncRoomPoolElement,empty  := sync.getPoolElementById(roomId)
	if empty {
		zlib.ExitPrint("syncRoomPoolElement is empty!!!")
	}
	if mynetWay.Option.LockMode == LOCK_MODE_PESSIMISTIC{
		if syncRoomPoolElement.PlayersAckStatus == PLAYERS_ACK_STATUS_WAIT{
			mylog.Error("syncRoomPoolElement PlayersAckStatus = ",PLAYERS_ACK_STATUS_WAIT,syncRoomPoolElement.PlayersAckList)
			zlib.ExitPrint(11111)
		}
	}
	PlayersAckList := make(map[int32]int32)
	for _,player:= range syncRoomPoolElement.PlayerList {
		if player.Status == PLAYER_STATUS_OFFLINE{
			mylog.Error("player offline")
			continue
		}
		//mynetWay.SendMsgByUid(player.Id,action,content)
		mynetWay.SendMsgCompressByUid(player.Id,action,contentStruct)
		PlayersAckList[player.Id] = 0
	}

	if mynetWay.Option.LockMode == LOCK_MODE_PESSIMISTIC{
		syncRoomPoolElement.PlayersAckList = PlayersAckList
		sync.upSyncRoomPoolElementPlayersAckStatus(roomId,PLAYERS_ACK_STATUS_WAIT)
	}
	content,_ := json.Marshal(contentStruct)
	sync.addOneRoomHistory(syncRoomPoolElement,action,string(content))

	//if testLock == 1{
	//	zlib.MyPrint(contentStruct)
	//	zlib.ExitPrint(3333)
	//}
}
func (sync *Sync)addOneRoomHistory(room *Room,action,content string){
	logicFrameHistory := ResponseRoomHistory{
		Action: action,
		Content: content,
	}
	//该局副本的所有玩家操作日志，用于断线重连-补放/重播
	room.LogicFrameHistory = append(room.LogicFrameHistory,&logicFrameHistory)
}
//一个房间的玩家的所有操作记录，一般用于C端断线重连时，恢复
func  (sync *Sync)RoomHistory(requestRoomHistory RequestRoomHistory,wsConn *WsConn){
	roomId := requestRoomHistory.RoomId
	room,_ := sync.getPoolElementById(roomId)
	responsepushRoomHistory := ResponsePushRoomHistory{}
	responsepushRoomHistory.List = room.LogicFrameHistory
	//responseRoomHistory := room.LogicFrameHistory
	//str,_ := json.Marshal(responseRoomHistory)
	//mynetWay.SendMsgByUid(wsConn.PlayerId,"pushRoomHistory",string(str))
}
//玩家掉线了，重新连接后，恢复游戏了~这个时候，要通知另外的玩家
func  (sync *Sync)playerResumeGame(requestPlayerResumeGame RequestPlayerResumeGame,wsConn *WsConn){
	//roomId := requestPlayerResumeGame.RoomId
	//str,_ := json.Marshal(requestPlayerResumeGame)
	//mynetWay.SendMsgByUid(wsConn.PlayerId,"otherPlayerResumeGame",string(str))
	//sync.boardCastInRoom(roomId,"otherPlayerResumeGame",string(str))
	responseOtherPlayerResumeGame := ResponseOtherPlayerResumeGame{
		PlayerId:requestPlayerResumeGame.PlayerId,
		SequenceNumber:requestPlayerResumeGame.SequenceNumber,
		RoomId:requestPlayerResumeGame.RoomId,
	}
	mynetWay.SendMsgCompressByUid(wsConn.PlayerId,"otherPlayerResumeGame",&responseOtherPlayerResumeGame)

}
//C端获取一个房间的信息
func  (sync *Sync)GetRoom(requestGetRoom RequestGetRoom,wsConn *WsConn){
	roomId := requestGetRoom.RoomId
	room,_ := sync.getPoolElementById(roomId)
	ResponsePushRoomInfo := ResponsePushRoomInfo{
		Id:room.Id,
		SequenceNumber: int32( room.SequenceNumber),
		AddTime: room.AddTime,
		PlayerList: room.PlayerList,
		Status :room.Status,
		Timeout: room.Timeout,
		RandSeek:room.RandSeek,
		RoomId: room.Id,
	}
	mynetWay.SendMsgCompressByUid(wsConn.PlayerId,"pushRoomInfo",&ResponsePushRoomInfo)
}