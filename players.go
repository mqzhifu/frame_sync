package main

import (
	"encoding/json"
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
//返回值是roomId,其实只是标注，让C端来获取
func  (players *Players)addPlayerPool(id int)string{
	mylog.Info("addPlayerPool :",id)
	hasPlayer,exist  := PlayerPool[id]
	if exist{
		mylog.Notice("this player id has exist pool")
		if hasPlayer.Status == PLAYER_STATUS_ONLINE{
			mylog.Error("hasPlayer.Status = PLAYER_STATUS_ONLINE")
		}else{
			players.upPlayerPool(id,PLAYER_STATUS_ONLINE)
		}
		return hasPlayer.RoomId
	}else{
		player := Player{
			Id: id,
			AddTime: zlib.GetNowTimeSecondToInt(),
			Nickname: "",
			Status: PLAYER_STATUS_ONLINE,
		}
		PlayerPool[id] = &player
	}
	return ""
}
func  (players *Players)getPlayerStatusById(plyaerId int)(player *Player,empty bool){
	playerStatus ,ok:= PlayerPool[plyaerId]
	if !ok{
		return player,true
	}
	return playerStatus,false
}

func  (players *Players)getPlayerStatus(requestPlayerStatus RequestPlayerStatus,wsConn *WsConn){
	player,_ := players.getPlayerStatusById(requestPlayerStatus.PlayerId)
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

func   (players *Players)upPlayerPool(id int,status int){

	player := PlayerPool[id]

	mylog.Info("upPlayerPool" , " old : ",player.Status," new:",status)

	player.Status = PLAYER_STATUS_ONLINE
	player.UpTime = zlib.GetNowTimeSecondToInt()

}
