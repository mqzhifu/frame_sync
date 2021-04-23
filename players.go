package main

import (
	"encoding/json"
	"errors"
	"zlib"
)

type Players struct {

}

type Player struct {
	Id 			int		`json:"id"`
	Nickname	string	`json:"nickname"`
	Status 		int		`json:"status"`
	AddTime		int		`json:"addTime"`
	UpTime 		int		`json:"upTime"`
	RoomId 		string	`json:"roomId"`
}
var PlayerPool map[int]*Player	//玩家 状态池
func PlayersNew()*Players{
	players := new(Players)
	PlayerPool = make(map[int]*Player)
	return players
}
func  (players *Players)addPlayerPool(id int)(existPlayer Player,err error){
	mylog.Info("addPlayerPool :",id)
	hasPlayer,empty  := players.getById(id)
	if !empty{
		mylog.Notice("this player id has exist pool",id)
		if hasPlayer.Status == PLAYER_STATUS_ONLINE{
			errMsg := "hasPlayer.Status = PLAYER_STATUS_ONLINE "
			mylog.Error(errMsg)
			err = errors.New(errMsg)
			return *hasPlayer,err
		}else{
			players.upPlayerStatus(id,PLAYER_STATUS_ONLINE)
			return *hasPlayer,nil
		}
	}else{
		mylog.Info("new player add")
		player := Player{
			Id: id,
			AddTime: zlib.GetNowTimeSecondToInt(),
			Nickname: "",
			Status: PLAYER_STATUS_ONLINE,
		}
		PlayerPool[id] = &player
		return player,nil
	}
	return existPlayer,nil
}
func  (players *Players)getById(playerId int)(player *Player,empty bool){
	myPlayer ,ok := PlayerPool[playerId]
	if ok {
		return myPlayer ,false
	}else{
		return player,true
	}
}

func  (players *Players)getPlayerStatus(requestPlayerStatus RequestPlayerStatus,wsConn *WsConn){
	player,_ := players.getById(requestPlayerStatus.PlayerId)
	responsePlayerStatus := ResponsePlayerStatus{
		Id :player.Id,
		Nickname: player.Nickname,
		Status : player.Status,
		AddTime: player.AddTime,
		UpTime 	: player.UpTime,
		RoomId 	: player.RoomId,
	}
	responsePlayerStatusStr,_ := json.Marshal(responsePlayerStatus)
	mynetWay.SendMsgByUid(wsConn.PlayerId,"pushPlayerStatus",string(responsePlayerStatusStr))
}
func  (players *Players)delById(playerId int){
	delete(PlayerPool,playerId)
}
func   (players *Players)upPlayerStatus(id int,status int){
	player := PlayerPool[id]

	mylog.Info("upPlayerStatus" , " old : ",player.Status," new:",status)

	player.Status = status
	player.UpTime = zlib.GetNowTimeSecondToInt()

}

func   (players *Players)upPlayerRoomId(playerId int,roomId string){
	player := PlayerPool[playerId]

	mylog.Info("upPlayerRoomId" , " old : ",player.RoomId," new:",roomId)

	player.RoomId = roomId
	player.UpTime = zlib.GetNowTimeSecondToInt()

}
