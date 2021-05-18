package netway

import (
	"encoding/json"
	"errors"
	"frame_sync/myproto"
	"strconv"
	"time"
	"zlib"
)

type Sync struct {
	Options Options
}

type Options struct {
	FPS 				int32 		`json:"fps"`			//frame pre second
	LockMode  			int32 		`json:"lockMode"`		//锁模式，乐观|悲观
	MapSize				int32		`json:"mapSize"`		//地址大小，给前端初始化使用
	Log *zlib.Log
}
//断点调试
var debugBreakPoint int
//集合 ：房间(一局游戏副本)
var MySyncRoomPool map[string]*Room
//索引表，PlayerId=>RoomId
var mySyncPlayerRoom map[int32]string
//同步 - 逻辑中的自增ID - 默认值
var logicFrameMsgDefaultId int32
//同步 - 逻辑中 - 操作帧的 自增ID - 默认值
var operationDefaultId int32
var RoomSyncMetricsPool map[string]RoomSyncMetrics

func NewSync(options Options)*Sync {
	mylog = options.Log
	mylog.Info("NewSync instance")
	MySyncRoomPool = make(map[string]*Room)
	mySyncPlayerRoom = make(map[int32]string)
	sync := new(Sync)
	sync.Options = options

	debugBreakPoint = 0
	logicFrameMsgDefaultId = 16
	operationDefaultId = 32
	//统计
	RoomSyncMetricsPool = make(map[string]RoomSyncMetrics)

	return sync
}


//给集合添加一个新的 游戏副本
func (sync *Sync) AddPoolElement(room 	*Room){
	mylog.Info("addPoolElement")
	_ ,exist := MySyncRoomPool[room.Id]
	if exist{
		mylog.Error("mySyncRoomPool has exist : ",room.Id)
		return
	}
	MySyncRoomPool[room.Id] = room
}
//进入战后，场景渲染完后，进入准确状态
func (sync *Sync)PlayerReady(requestPlayerReady myproto.RequestPlayerReady,wsConn *WsConn) {
	//zlib.MyPrint(requestPlayerReady.PlayerId)
	//zlib.ExitPrint(requestPlayerReady)
	roomId := mySyncPlayerRoom[requestPlayerReady.PlayerId]
	mylog.Debug("mySyncPlayerRoom:", mySyncPlayerRoom, " , id :",roomId)
	room,empty := sync.getPoolElementById(roomId)
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
	responseStartBattle := myproto.ResponseStartBattle{
		SequenceNumberStart: int32(0),
	}
	sync.boardCastFrameInRoom(room.Id,"startBattle",&responseStartBattle)
	room.UpStatus(ROOM_STATUS_EXECING)
	room.StartTime = int32(zlib.GetNowTimeSecondToInt())


	RoomSyncMetricsPool[roomId] = RoomSyncMetrics{}

	sync.testFirstLogicFrame(room)
	room.ReadyCloseChan <- 1
	//开启定时器，推送逻辑帧
	go sync.logicFrameLoop(room)

}
func  (sync *Sync)testFirstLogicFrame(room *Room){
	//初始结束后，这里方便测试，再补一帧，所有玩家的随机位置
	if room.PlayerList[0].Id < 999{
		var operations []*myproto.Operation
		for _,player:= range room.PlayerList{
			location := strconv.Itoa(zlib.GetRandInt32Num(sync.Options.MapSize)) + "," + strconv.Itoa(zlib.GetRandInt32Num(sync.Options.MapSize))
			operation := myproto.Operation{
				Id:       logicFrameMsgDefaultId,
				Event:    "move",
				Value:    location,
				PlayerId: player.Id,
			}
			operations = append(operations,&operation)
		}
		logicFrameMsg := myproto.ResponsePushLogicFrame{
			Id	:             operationDefaultId,
			RoomId:             room.Id,
			SequenceNumber :    int32(room.SequenceNumber),
			Operations 		: operations,
		}
		sync.boardCastInRoom(room.Id,"pushLogicFrame",&logicFrameMsg)
	}
}
//根据playerId 反查 roomId
func  (sync *Sync)GetMySyncPlayerRoomById(playerId int32)string{
	roomId,_ :=  mySyncPlayerRoom[playerId]
	return roomId
}
//检查 所有玩家是否 都已准确，超时了
func  (sync *Sync)checkReadyTimeout(room *Room){
	for{
		select {
			case   <-room.ReadyCloseChan:
				goto end
			default:
				now := zlib.GetNowTimeSecondToInt()
				if now > int(room.ReadyTimeout){
					mylog.Error("room ready timeout id :",room.Id)
					requestReadyTimeout := myproto.ResponseReadyTimeout{
						RoomId: room.Id,
					}
					sync.boardCastInRoom(room.Id,"readyTimeout",&requestReadyTimeout)
					sync.roomEnd(room.Id,0)
					goto end
				}
				time.Sleep(time.Second * 1)
		}
	}
end:
	mylog.Warning("checkReadyTimeout loop routine close")
}
//检查 所有玩家是否 都已准确，超时了
func  (sync *Sync)playerOfflineCheckRoomTimeout(room *Room){
	for{
		select {
			case   <-room.ReadyCloseChan:
				goto end
			default:
				now := zlib.GetNowTimeSecondToInt()
				if now > int(room.ReadyTimeout){
					mylog.Error("room ready timeout id :",room.Id)
					requestReadyTimeout := myproto.ResponseReadyTimeout{
						RoomId: room.Id,
					}
					sync.boardCastInRoom(room.Id,"readyTimeout",&requestReadyTimeout)
					sync.roomEnd(room.Id,0)
					goto end
				}
				time.Sleep(time.Second * 1)
		}
	}
end:
	mylog.Warning("playerOfflineCheckRoomTimeout loop routine close")
}

//一局新游戏，均已准备，进入开始状态
func  (sync *Sync)Start(roomId string){
	mylog.Warning("start a new game:")

	room,_ := sync.getPoolElementById(roomId)
	mylog.Warning("roomInfo:",room)
	room.UpStatus(ROOM_STATUS_READY)
	responseClientInitRoomData := myproto.ResponseEnterBattle{
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

	sync.boardCastInRoom(roomId,"enterBattle",&responseClientInitRoomData)
	room.CloseChan = make(chan int)
	room.ReadyCloseChan = make(chan int)
	go sync.checkReadyTimeout(room)
}
//在一个房间内，搜索一个用户
func (sync *Sync)getPlayerByIdInRoom(playerId int32 ,room *Room,)(myplayer *myproto.Player,empty bool){
	for _,player:= range room.PlayerList{
		if player.Id == playerId{
			return player,false
		}
	}
	return myplayer,true
}
//根据ROOID  有池子里找到该roomInfo
func (sync *Sync) getPoolElementById(roomId string)(SyncRoomPoolElement *Room,empty bool){
	v,exist := MySyncRoomPool[roomId]
	if !exist{
		mylog.Error("getPoolElementById is empty,",roomId)
		return SyncRoomPoolElement,true
	}
	return v,false
}
//同步 玩家 操作 定时器
func  (sync *Sync)logicFrameLoop(room *Room){
	if sync.Options.FPS > 1000 {
		zlib.ExitPrint("fps > 1000 ms")
	}

	fpsTime :=  1000 /  sync.Options.FPS
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
//同上
func  (sync *Sync)logicFrameLoopReal(room *Room,fpsTime int32)int32{
	queue := room.PlayersOperationQueue
	end := queue.Len()
	if end <= 0 {
		return fpsTime
	}

	if sync.Options.LockMode == LOCK_MODE_PESSIMISTIC {
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

	logicFrame := myproto.ResponsePushLogicFrame{
		Id:             0,
		RoomId:         room.Id,
		SequenceNumber: int32(room.SequenceNumber),
	}
	var operations []*myproto.Operation
	i := 0
	element := queue.Front()

	for {
		if i >= end {
			break
		}
		operationsValueInterface := element.Value
		operationsValue := operationsValueInterface.(string)
		var elementOperations []myproto.Operation
		err := json.Unmarshal([]byte(operationsValue), &elementOperations)
		if err != nil {
			mylog.Error("queue json.Unmarshal err :", err.Error())
		}
		//mylog.Debug(operationsValue,"elementOperations",elementOperations)
		for j := 0; j < len(elementOperations); j++ {
			//if elementOperations[j].Event != "empty"{
			//	mylog.Debug("elementOperations j :",elementOperations[j])
			//	debugBreakPoint = 1
			//}
			operations = append(operations, &elementOperations[j])
		}



		tmpElement := element.Next()
		queue.Remove(element)
		element = tmpElement

		i++
	}
	mylog.Info("operations:",operations)
	logicFrame.Operations = operations
	sync.boardCastFrameInRoom(room.Id, "pushLogicFrame",&logicFrame)
	return fpsTime
}
//定时，接收玩家的操作记录
func  (sync *Sync)ReceivePlayerOperation(logicFrame myproto.RequestPlayerOperations,wsConn *WsConn,content string){
	//mylog.Debug(logicFrame)
	room,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mylog.Error("getPoolElementById is empty",logicFrame.RoomId)
	}
	err := sync.checkReceiveOperation(room,logicFrame,wsConn)
	if err != nil{
		mylog.Error("receivePlayerOperation check error:",err.Error())
		return
	}
	if len(logicFrame.Operations) < 0{
		mylog.Error("len(logicFrame.Operations) < 0")
		return
	}
	//roomSyncMetrics := roomSyncMetricsPool[logicFrame.RoomId]
	//roomSyncMetrics.InputNum ++
	//roomSyncMetrics.InputSize = roomSyncMetrics.InputSize + len(content)

	logicFrameStr ,_ := json.Marshal(logicFrame.Operations)
	room.PlayersOperationQueue.PushBack(string(logicFrameStr))
	if room.PlayersAckList[wsConn.PlayerId] == 0{
		room.PlayersAckList[wsConn.PlayerId] = 1
	}
}
//检测玩家发送的操作是否合规
func  (sync *Sync)checkReceiveOperation(room *Room,logicFrame myproto.RequestPlayerOperations,wsConn *WsConn)error{
	if room.Status == ROOM_STATUS_INIT {
		return errors.New("room status err is  ROOM_STATUS_INIT  "+strconv.Itoa(int(room.Status)))
	}else if room.Status == ROOM_STATUS_END {
		return errors.New("room status err is ROOM_STATUS_END  "+strconv.Itoa(int(room.Status)))
	}else if room.Status == ROOM_STATUS_PAUSE {
		//暂时状态，囚徒模式下
		//当A掉线后，会立刻更新房间状态为:暂停，但是其它未掉线的玩家依然还会发当前帧的操作数据
		//此时，房间已进入暂停状态，如果直接拒掉该条消息，会导致A恢复后，发送当前帧数据是正常的
		//而，其它玩家因为消息被拒，导致此条消息只有A发送成功，但是迟迟等不到其它玩家再未发送消息，该帧进入死锁
		//固，这里做出改变，暂停状态下：正常玩家可以多发一帧，等待掉线玩家重新上线
		if int(logicFrame.SequenceNumber) == room.SequenceNumber{
			mylog.Warning("logicFrame.SequenceNumber  == room.SequenceNumber")
			//只有掉线的玩家，最后这一帧的数据没有发出来，才会到这个条件里
			//但，其它正常玩家如果还是一直不停的在发 这一帧，QUEUE 就爆了
			if room.PlayersAckList[wsConn.PlayerId] == 1{
				msg := "(offline) last frame Players has ack ,don'send... "
				mylog.Error(msg)
				return errors.New(msg)
			}else{

			}
		}else{
			c_n := strconv.Itoa(int(logicFrame.SequenceNumber))
			r_n := strconv.Itoa(int( room.SequenceNumber))
			msg := "room status is ROOM_STATUS_PAUSE ,on receive num   c_n"+c_n + " ,r_n : "+r_n
			return errors.New( msg )
		}

	}else if room.Status == ROOM_STATUS_EXECING {

	}else{
		return errors.New("room status num error.  "+strconv.Itoa(int(room.Status)))
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
//游戏结束
func  (sync *Sync)roomEnd(roomId string,sendCloseChan int){
	mylog.Info("roomEnd")
	room,empty := sync.getPoolElementById(roomId)
	if empty{
		mylog.Error("getPoolElementById is empty",roomId)
		return
	}
	room.UpStatus(ROOM_STATUS_END)
	room.EndTime = int32(zlib.GetNowTimeSecondToInt())
	for _,v:= range room.PlayerList{
		mynetWay.Players.UpPlayerRoomId(v.Id,"")
		delete(mySyncPlayerRoom,v.Id)
	}
	if sendCloseChan == 1{
		room.CloseChan <- 1
	}
}
//玩家操作后，触发C端主动发送游戏结束事件
func  (sync *Sync)GameOver(requestGameOver myproto.RequestGameOver,wsConn *WsConn){
	responseGameOver := myproto.ResponseGameOver{
		PlayerId : requestGameOver.PlayerId,
		RoomId:requestGameOver.RoomId,
		SequenceNumber: requestGameOver.SequenceNumber,
		Result:requestGameOver.Result,
	}
	sync.boardCastInRoom(requestGameOver.RoomId,"gameOver",&responseGameOver)

	sync.roomEnd(requestGameOver.RoomId,1)
}
//玩家触发了该角色死亡
func  (sync *Sync)PlayerOver(requestGameOver myproto.RequestPlayerOver,wsConn *WsConn){
	roomId := mySyncPlayerRoom[requestGameOver.PlayerId]
	responseOtherPlayerOver := myproto.ResponseOtherPlayerOver{PlayerId: requestGameOver.PlayerId}
	sync.boardCastInRoom(roomId,"otherPlayerOver",&responseOtherPlayerOver)
}
//更新一个逻辑帧的确认状态
func (sync *Sync)upSyncRoomPoolElementPlayersAckStatus(roomId string,status int){
	syncRoomPoolElement,_  := sync.getPoolElementById(roomId)
	mylog.Notice("upSyncRoomPoolElementPlayersAckStatus ,old :",syncRoomPoolElement.PlayersAckStatus,"new",status)
	syncRoomPoolElement.PlayersAckStatus = status
}
//判定一个房间内，玩家在线的人
func (sync *Sync)roomOnlinePlayers(room *Room)[]int32{
	var playerOnLine []int32
	for _, v := range room.PlayerList {
		player, empty := mynetWay.Players.GetById(v.Id)
		//mylog.Debug("pinfo::",player," empty:",empty," ,pid:",v.Id)
		if empty {
			continue
		}
		//zlib.MyPrint(player.Status)
		if player.Status == PLAYER_STATUS_ONLINE {
			zlib.MyPrint("playerOnLine append")
			playerOnLine = append(playerOnLine,player.Id)
		}
	}
	//zlib.MyPrint(playerOnLine)
	return playerOnLine
}
//玩家断开连接后
func (sync *Sync)Close(wsConn *WsConn){
	mylog.Warning("sync.close")
	//获取该玩家的roomId
	roomId,ok := mySyncPlayerRoom[wsConn.PlayerId]
	if !ok || roomId == "" {
		return
	}
	//判断下所有玩家是否均下线了
	room, _ := sync.getPoolElementById(roomId)
	if room.Status == ROOM_STATUS_EXECING || room.Status == ROOM_STATUS_PAUSE {
		playerOnLine := sync.roomOnlinePlayers(room)
		//mylog.Debug("playerOnLine:",playerOnLine, "len :",len(playerOnLine))
		playerOnLineCount := len(playerOnLine)
		//playerOnLineCount-- //这里因为，已有一个玩家关闭中，但是还未处理
		mylog.Info("ROOM_STATUS_EXECING ,has check roomEnd , playerOnLineCount : ", playerOnLineCount)
		if playerOnLineCount <= 0 {
			//
			sync.roomEnd(roomId,1)
		}else{
			if room.Status == ROOM_STATUS_EXECING {
				room.UpStatus(ROOM_STATUS_PAUSE)
				responseOtherPlayerOffline := myproto.ResponseOtherPlayerOffline{
					PlayerId: wsConn.PlayerId,
				}
				sync.boardCastInRoom(roomId,"otherPlayerOffline",&responseOtherPlayerOffline)
			}
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
		if player.Status == PLAYER_STATUS_OFFLINE {
			mylog.Error("player offline")
			continue
		}
		//mynetWay.SendMsgByUid(player.Id,action,content)
		mynetWay.SendMsgCompressByUid(player.Id,action,contentStruct)
	}
	//content ,_:= json.Marshal(contentStruct)
	content ,_ := json.Marshal(JsonCamelCase{contentStruct})
	sync.addOneRoomHistory(room,action,string(content))
}
//给一个副本里的所有玩家广播数据，且该数据必须得有C端ACK
func  (sync *Sync)boardCastFrameInRoom(roomId string,action string ,contentStruct interface{}){
	mylog.Notice("boardCastFrameInRoom:",roomId,action)
	syncRoomPoolElement,empty  := sync.getPoolElementById(roomId)
	if empty {
		zlib.ExitPrint("syncRoomPoolElement is empty!!!")
	}
	if sync.Options.LockMode == LOCK_MODE_PESSIMISTIC {
		if syncRoomPoolElement.PlayersAckStatus == PLAYERS_ACK_STATUS_WAIT {
			mylog.Error("syncRoomPoolElement PlayersAckStatus = ", PLAYERS_ACK_STATUS_WAIT,syncRoomPoolElement.PlayersAckList)
			return
		}
	}
	PlayersAckList := make(map[int32]int32)
	for _,player:= range syncRoomPoolElement.PlayerList {
		PlayersAckList[player.Id] = 0
		if player.Status == PLAYER_STATUS_OFFLINE {
			mylog.Error("player offline")
			continue
		}
		//mynetWay.SendMsgByUid(player.Id,action,content)
		mynetWay.SendMsgCompressByUid(player.Id,action,contentStruct)

	}

	if sync.Options.LockMode == LOCK_MODE_PESSIMISTIC {
		syncRoomPoolElement.PlayersAckList = PlayersAckList
		sync.upSyncRoomPoolElementPlayersAckStatus(roomId, PLAYERS_ACK_STATUS_WAIT)
	}
	//content,_ := json.Marshal(contentStruct)
	content ,_ := json.Marshal(JsonCamelCase{contentStruct})
	sync.addOneRoomHistory(syncRoomPoolElement,action,string(content))

	//if debugBreakPoint == 1{
	//	zlib.MyPrint(contentStruct)
	//	zlib.ExitPrint(3333)
	//}
}
func (sync *Sync)addOneRoomHistory(room *Room,action,content string){
	logicFrameHistory := myproto.ResponseRoomHistory{
		Action: action,
		Content: content,
	}
	//该局副本的所有玩家操作日志，用于断线重连-补放/重播
	room.LogicFrameHistory = append(room.LogicFrameHistory,&logicFrameHistory)
}
//一个房间的玩家的所有操作记录，一般用于C端断线重连时，恢复
func  (sync *Sync)RoomHistory(requestRoomHistory myproto.RequestRoomHistory,wsConn *WsConn){
	roomId := requestRoomHistory.RoomId
	room,_ := sync.getPoolElementById(roomId)
	responsePushRoomHistory := myproto.ResponsePushRoomHistory{}
	responsePushRoomHistory.List = room.LogicFrameHistory
	mynetWay.SendMsgCompressByUid(wsConn.PlayerId,"pushRoomHistory",&responsePushRoomHistory)
}
//玩家掉线了，重新连接后，恢复游戏了~这个时候，要通知另外的玩家
func  (sync *Sync)PlayerResumeGame(requestPlayerResumeGame myproto.RequestPlayerResumeGame,wsConn *WsConn){
	room ,empty := sync.getPoolElementById(requestPlayerResumeGame.RoomId)
	if empty{
		mylog.Error("playerResumeGame get room empty")
		return
	}
	var restartGame = 0
	var playerIds []int32
	if room.Status == ROOM_STATUS_PAUSE {
		playerOnlineNum := sync.roomOnlinePlayers(room)
		if  len(playerOnlineNum) == len(room.PlayerList){
			room.UpStatus(ROOM_STATUS_EXECING)
			restartGame = 1
			for _, v:= range room.PlayerList{
				playerIds = append(playerIds,v.Id)
			}
		}
	}

	responseOtherPlayerResumeGame := myproto.ResponseOtherPlayerResumeGame{
		PlayerId:requestPlayerResumeGame.PlayerId,
		SequenceNumber:requestPlayerResumeGame.SequenceNumber,
		RoomId:requestPlayerResumeGame.RoomId,
	}
	sync.boardCastInRoom(room.Id,"otherPlayerResumeGame",&responseOtherPlayerResumeGame)
	//mynetWay.SendMsgCompressByUid(wsConn.PlayerId,)
	if restartGame == 1{
		responseRestartGame := myproto.ResponseRestartGame{
			RoomId: requestPlayerResumeGame.RoomId,
			PlayerIds: playerIds,
		}
		sync.boardCastInRoom(room.Id,"restartGame",&responseRestartGame)
	}

}
//C端获取一个房间的信息
func  (sync *Sync)GetRoom(requestGetRoom myproto.RequestGetRoom,wsConn *WsConn){
	roomId := requestGetRoom.RoomId
	room,_ := sync.getPoolElementById(roomId)
	ResponsePushRoomInfo := myproto.ResponsePushRoomInfo{
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