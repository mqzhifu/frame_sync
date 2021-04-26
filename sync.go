package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"zlib"
)

type Sync struct {

}
//集合中的：每个副本元素
//type SyncRoomPoolElement struct {
//	Status 				int
//	Room 				*Room
//	SequenceNumber		int
//	PlayersAckList		map[int]int
//	PlayersAckStatus	int
//	AddTime 			int
//	RandSeek			int
//	LogicFrameHistory 	[]LogicFrameHistory
//}
type LogicFrameHistory struct {
	Id 		int
	Action 	string
	Content string
}
//逻辑帧
type LogicFrame struct {
	Id 				int 		`json:"id"`
	RoomId 			string		`json:"roomId"`
	SequenceNumber 	int			`json:"sequenceNumber"`
	Commands 		[]Command	`json:"commands"`
}
//玩家操作指令集
type Command struct {
	Id 			int 	`json:"id"`
	Action 		string	`json:"action"`
	Value 		string	`json:"value"`
	PlayerId 	int		`json:"playerId"`
}
//集合 ：每个副本
var mySyncRoomPool map[string]*Room
//索引表，PlayerId=>RoomId
var mySyncPlayerRoom map[int]string
var logicFrameMsgDefaultId int
var commandDefaultId int
func NewSync()*Sync{
	mylog.Info("NewSync instance")
	mySyncRoomPool = make(map[string]*Room)
	mySyncPlayerRoom = make(map[int]string)
	sync := new(Sync)

	logicFrameMsgDefaultId = 16
	commandDefaultId = 32

	return sync
}
func (sync *Sync)checkRoomTimeoutLoop(){
	for{
		select {
		default:
			sync.checkRoomTimeout()
		}
	}
}
func (sync *Sync)checkRoomTimeout(){
	roomLen := len(mySyncRoomPool)
	if roomLen <= 0{
		return
	}

	now :=  zlib.GetNowTimeSecondToInt()
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
//一个新的房间，开始同步
func  (sync *Sync)start(roomId string){
	mynetWay.Option.Mylog.Warning("start a new game:")

	room,_ := sync.getPoolElementById(roomId)
	room.upStatus(ROOM_STATUS_EXECING)
	responseClientInitRoomData := ResponseClientInitRoomData{
		Status 			:room.Status,
		AddTime 		:room.AddTime,
		RoomId			:roomId,
		SequenceNumber	: 0,
		PlayerList		:room.PlayerList,
		RandSeek		:room.RandSeek,
		Time			:time.Now().UnixNano() / 1e6,
	}

	for _,v :=range room.PlayerList{
		mySyncPlayerRoom[v.Id] = roomId
	}

	jsonStrByte,_ := json.Marshal(responseClientInitRoomData)
	sync.boardCastInRoom(roomId,"startInit",string(jsonStrByte))

	//初始结束后，这里方便测试，再补一帧，所有玩家的随机位置
	if room.PlayerList[0].Id < 999{
		commands := []Command{}
		for _,player:= range room.PlayerList{
			location := strconv.Itoa(zlib.GetRandIntNum(mynetWay.Option.MapSize)) + "," + strconv.Itoa(zlib.GetRandIntNum(mynetWay.Option.MapSize))
			command := Command{
				Id: logicFrameMsgDefaultId,
				Action: "move",
				Value: location,
				PlayerId: player.Id,
			}
			commands = append(commands,command)
		}
		logicFrameMsg := LogicFrame{
			Id	: commandDefaultId,
			RoomId: roomId,
			SequenceNumber :room.SequenceNumber,
			Commands 		:commands,
		}
		logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
		sync.boardCastInRoom(roomId,"pushLogicFrame",string(logicFrameMsgJson))

	}

	room.CloseChan = make(chan int)
	go sync.pushLogicFrame(room)
}
//有一个房间内，搜索一个用户
func (sync *Sync)getPlayerByIdInRoom(playerId int ,room *Room,)(myplayer *Player,empty bool){
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
func  (sync *Sync)createOneCommandQueue(room Room){
	//queue := list.New()
	//commandQueuePool[room.Id] = queue
}
func  (sync *Sync)pushLogicFrame(room *Room){
	for{
		select {
		case   <-room.CloseChan:
			goto end
		default:
			sync.pushLogicFrameReal(room)
			sleepMsTime := sync.pushLogicFrameReal(room)
			sleepMsTimeD := time.Duration(sleepMsTime)
			if sleepMsTime > 0 {
				time.Sleep(sleepMsTimeD * time.Millisecond)
			}
		}
	}
end:
	mylog.Warning("pushLogicFrame loop routine close")
}

func  (sync *Sync)pushLogicFrameReal(room *Room)int{
	queue := room.CommandQueue
	end := queue.Len()
	//mylog.Info("queue.Len:", end)
	if end <= 0 {
		//mylog.Warning("commandQueue len == 0")
		return 100
	}

	ack := 0
	for _, v := range room.PlayersAckList {
		if v == 1 {
			ack++
		}
	}

	if ack < len(room.PlayersAckList) {
		mylog.Error("还有玩家未发送操作记录")
		return 100
	}

	sync.upSyncRoomPoolElementPlayersAckStatus(room.Id, PLAYERS_ACK_STATUS_OK)
	room.SequenceNumber++

	logicFrame := LogicFrame{
		Id:             0,
		RoomId:         room.Id,
		SequenceNumber: room.SequenceNumber,
	}
	commands := []Command{}
	i := 0
	element := queue.Front()
	for {
		//mylog.Info("pushLogicFrame i : ",i)
		if i >= end {
			break
		}
		commandValueInterface := element.Value
		commandValue := commandValueInterface.(string)
		var elementCommands []Command
		err := json.Unmarshal([]byte(commandValue), &elementCommands)
		if err != nil {
			mylog.Error("json.Unmarshal err :", err.Error())
		}
		for j := 0; j < len(elementCommands); j++ {
			commands = append(commands, elementCommands[j])
		}

		tmpElement := element.Next()
		queue.Remove(element)
		element = tmpElement

		i++
	}

	logicFrame.Commands = commands
	logicFrameStr, _ := json.Marshal(logicFrame)
	sync.boardCastFrameInRoom(room.Id, "pushLogicFrame", string(logicFrameStr))

	return 100
}

func  (sync *Sync)receiveCommand(logicFrame LogicFrame,wsConn *WsConn){
	room,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",logicFrame.RoomId)
	}
	err := sync.checkReceiveCommand(room,logicFrame,wsConn)
	if err != nil{
		mylog.Error("receiveCommand check error:",err.Error())
		return
	}
	room.PlayersAckList[wsConn.PlayerId] = 1
	logicFrameStr ,_ := json.Marshal(logicFrame.Commands)
	room.CommandQueue.PushBack(string(logicFrameStr))
	room.PlayersAckList[wsConn.PlayerId] = 1
}
func  (sync *Sync)checkReceiveCommand(room *Room,logicFrame LogicFrame,wsConn *WsConn)error{
	if room.Status == ROOM_STATUS_INIT{
		return errors.New("room status err,init......")
	}

	if room.Status == ROOM_STATUS_END{
		return errors.New("room status err,has end")
	}
	if logicFrame.SequenceNumber != room.SequenceNumber{
		return errors.New("room.SequenceNumber err")
	}
	return nil
}

func  (sync *Sync)receiveCommandOld(logicFrame LogicFrame,wsConn *WsConn){
	room,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("getPoolElementById is empty",logicFrame.RoomId)
	}
	err := sync.checkReceiveCommand(room,logicFrame,wsConn)
	if err != nil{
		mylog.Error(err.Error())
		return
	}
	logicFrame.SequenceNumber = room.SequenceNumber
	logicFrameMsgJson ,_ := json.Marshal(logicFrame)
	sync.boardCastFrameInRoom(logicFrame.RoomId,"pushLogicFrame",string(logicFrameMsgJson))
}

func (sync *Sync) playerLogicFrameAck(logicFrame LogicFrame,wsConn *WsConn ){
	mynetWay.Option.Mylog.Info("playerLogicFrameAck",wsConn.PlayerId)

	syncRoomPoolElement,empty := sync.getPoolElementById(logicFrame.RoomId)
	if empty{
		mynetWay.Option.Mylog.Error("roomId is wrong",logicFrame.RoomId)
		return
	}
	if syncRoomPoolElement.SequenceNumber != logicFrame.SequenceNumber{
		mynetWay.Option.Mylog.Error("SequenceNumber wrong",logicFrame.SequenceNumber,syncRoomPoolElement.SequenceNumber)
		return
	}
	if syncRoomPoolElement.PlayersAckStatus != PLAYERS_ACK_STATUS_WAIT{
		mynetWay.Option.Mylog.Error("该帧号的状态!=待确认 状态,当前状态为：",syncRoomPoolElement.PlayersAckStatus)
		return
	}

	player ,empty := sync.getPlayerByIdInRoom(wsConn.PlayerId,syncRoomPoolElement)
	if empty{
		mynetWay.Option.Mylog.Error("playerId wrong,not found in this room.",wsConn.PlayerId)
		return
	}
	if syncRoomPoolElement.PlayersAckList[player.Id] == 1{
		mynetWay.Option.Mylog.Error("该玩家已经确认过了，不要重复操作")
		return
	}else{
		syncRoomPoolElement.PlayersAckList[player.Id] = 1
	}

	ack := 0
	for _,v := range syncRoomPoolElement.PlayersAckList{
		if v == 1{
			ack++
		}
	}
	mynetWay.Option.Mylog.Info("this room LogicFrameSequenceNumber(",syncRoomPoolElement.SequenceNumber,") ack total:",ack)

	//该逻辑帧已全部确认
	if ack == len(syncRoomPoolElement.PlayersAckList){
		mynetWay.Option.Mylog.Info("one logic frame all ack.")
		//更新 帧号 变更状态
		mylog.Info("syncRoomPoolElement.SequenceNumber inc: old",syncRoomPoolElement.SequenceNumber)
		syncRoomPoolElement.SequenceNumber++
		//mylog.Info("syncRoomPoolElement.SequenceNumber inc: new",syncRoomPoolElement.SequenceNumber)
		syncRoomPoolElement.PlayersAckList = make(map[int]int)
		sync.upSyncRoomPoolElementPlayersAckStatus(logicFrame.RoomId,PLAYERS_ACK_STATUS_INIT)
		//syncRoomPoolElement.PlayersAckStatus = PLAYERS_ACK_STATUS_INIT
		mylog.Notice("have a new SequenceNumber:",syncRoomPoolElement.SequenceNumber)
		//这里有个特殊处理，首帧其实是初始化的一些数据，当完成后，第一帧的动作按说应该是玩家操作
		//但这里，先模拟一下，将玩家随机散落到一地图上的一些点
		if syncRoomPoolElement.SequenceNumber == 1 {
			mynetWay.Option.Mylog.Info("syncRoomPoolElement.SequenceNumber == 0")
			//S端，每一逻辑帖，建立一个集合，保存广播的消息，玩家返回的确认ACK
			commands := []Command{}
			for _,player:= range syncRoomPoolElement.PlayerList{
				location := strconv.Itoa(zlib.GetRandIntNum(mynetWay.Option.MapSize)) + "," + strconv.Itoa(zlib.GetRandIntNum(mynetWay.Option.MapSize))
				command := Command{
					Id: logicFrameMsgDefaultId,
					Action: "move",
					Value: location,
					PlayerId: player.Id,
				}
				commands = append(commands,command)
			}
			logicFrameMsg := LogicFrame{
				Id	: commandDefaultId,
				RoomId: logicFrame.RoomId,
				SequenceNumber :syncRoomPoolElement.SequenceNumber,
				Commands 		:commands,
			}
			logicFrameMsgJson ,_ := json.Marshal(logicFrameMsg)
			sync.boardCastFrameInRoom(logicFrame.RoomId,"pushLogicFrame",string(logicFrameMsgJson))
		}
	}

	mynetWay.Option.Mylog.Info("playerLogicFrameAck~ finish.")
	return

}
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
	sync.roomEnd(requestGameOver.RoomId)
	jsonStr ,_ := json.Marshal(requestGameOver)
	sync.boardCastInRoom(requestGameOver.RoomId,"gameOver",string(jsonStr))
}

func (sync *Sync)upSyncRoomPoolElementPlayersAckStatus(roomId string,status int){
	syncRoomPoolElement,_  := sync.getPoolElementById(roomId)
	mylog.Notice("upSyncRoomPoolElementPlayersAckStatus ,old :",syncRoomPoolElement.PlayersAckStatus,"new",status)
	syncRoomPoolElement.PlayersAckStatus = status
}
//一个玩家取消了准备/报名
func (sync *Sync)cancelSign(requestCancelSign RequestCancelSign,wsConn *WsConn){
	myMatch.delOnePlayer(wsConn.PlayerId)
}
//判定一个房间内，玩家在线的人
func (sync *Sync)roomOnlinePlayers(room *Room)[]int{
	var playerOnLine []int
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
		mylog.Info("playerOnLineCount : ", playerOnLineCount)
		if playerOnLineCount <= 0 {
			sync.roomEnd(roomId)
		}
	}

	responseOtherPlayerOffline := ResponseOtherPlayerOffline{
		PlayerId: wsConn.PlayerId,
	}
	jsonStr,_ := json.Marshal(responseOtherPlayerOffline)
	sync.boardCastFrameInRoom(roomId,"otherPlayerOffline",string(jsonStr))
}
//单纯的给一个房间里的人发消息，不考虑是否有顺序号的情况
func  (sync *Sync)boardCastInRoom(roomId string,action string ,content string){
	room,empty  := sync.getPoolElementById(roomId)
	if empty {
		zlib.ExitPrint("syncRoomPoolElement is empty!!!")
	}
	for _,player:= range room.PlayerList {
		if player.Status == PLAYER_STATUS_OFFLINE{
			mylog.Error("player offline")
			continue
		}
		mynetWay.SendMsgByUid(player.Id,action,content)
	}

	sync.addOneRoomHistory(room,action,content)
}
//给一个副本里的所有玩家广播数据，且该数据必须得有C端ACK
func  (sync *Sync)boardCastFrameInRoom(roomId string,action string ,content string){
	mylog.Notice("boardCastFrameInRoom:",roomId,action)
	syncRoomPoolElement,empty  := sync.getPoolElementById(roomId)
	if empty {
		zlib.ExitPrint("syncRoomPoolElement is empty!!!")
	}
	if syncRoomPoolElement.PlayersAckStatus == PLAYERS_ACK_STATUS_WAIT{
		mylog.Error("syncRoomPoolElement PlayersAckStatus = ",PLAYERS_ACK_STATUS_WAIT,syncRoomPoolElement.PlayersAckList)
		zlib.ExitPrint(11111)
	}
	PlayersAckList := make(map[int]int)
	for _,player:= range syncRoomPoolElement.PlayerList {
		if player.Status == PLAYER_STATUS_OFFLINE{
			mylog.Error("player offline")
			continue
		}
		mynetWay.SendMsgByUid(player.Id,action,content)
		PlayersAckList[player.Id] = 0
	}

	syncRoomPoolElement.PlayersAckList = PlayersAckList
	sync.upSyncRoomPoolElementPlayersAckStatus(roomId,PLAYERS_ACK_STATUS_WAIT)

	sync.addOneRoomHistory(syncRoomPoolElement,action,content)
}
func (sync *Sync)addOneRoomHistory(room *Room,action,content string){
	logicFrameHistory := LogicFrameHistory{
		Action: action,
		Content: content,
	}
	//该局副本的所有玩家操作日志，用于断线重连-补放/重播
	room.LogicFrameHistory = append(room.LogicFrameHistory,logicFrameHistory)
}
//一个房间的玩家的所有操作记录，一般用于C端断线重连时，恢复
func  (sync *Sync)RoomHistory(requestRoomHistory RequestRoomHistory,wsConn *WsConn){
	roomId := requestRoomHistory.RoomId
	room,_ := sync.getPoolElementById(roomId)
	responseRoomHistory := room.LogicFrameHistory
	str,_ := json.Marshal(responseRoomHistory)
	mynetWay.SendMsgByUid(wsConn.PlayerId,"pushRoomHistory",string(str))
}
//玩家掉线了，重新连接后，恢复游戏了~这个时候，要通知另外的玩家
func  (sync *Sync)playerResumeGame(requestPlayerResumeGame RequestPlayerResumeGame,wsConn *WsConn){
	roomId := requestPlayerResumeGame.RoomId
	str,_ := json.Marshal(requestPlayerResumeGame)
	mynetWay.SendMsgByUid(wsConn.PlayerId,"otherPlayerResumeGame",string(str))
	sync.boardCastInRoom(roomId,"otherPlayerResumeGame",string(str))
}
//C端获取一个房间的信息
func  (sync *Sync)GetRoom(requestGetRoom RequestGetRoom,wsConn *WsConn){
	roomId := requestGetRoom.RoomId
	room,_ := sync.getPoolElementById(roomId)
	responseClientInitRoomData := ResponseClientInitRoomData{
		Status 			:room.Status,
		AddTime 		:room.AddTime,
		RoomId			:roomId,
		SequenceNumber	: 0,
		PlayerList		:room.PlayerList,
		RandSeek		:room.RandSeek,
		Time			:time.Now().UnixNano() / 1e6,
	}
	str,_ := json.Marshal(responseClientInitRoomData)
	mynetWay.SendMsgByUid(wsConn.PlayerId,"pushRoomInfo",string(str))
}
