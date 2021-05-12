package netway

import (
	"errors"
	"frame_sync/myproto"
	"zlib"
)

type Players struct {

}

//type Player struct {
//	Id 			int		`json:"id"`
//	RoleId 		int 	`json:"roleId"`
//	Nickname	string	`json:"nickname"`
//	Status 		int		`json:"status"`
//	AddTime		int		`json:"addTime"`
//	UpTime 		int		`json:"upTime"`
//	RoomId 		string	`json:"roomId"`
//	UDPPort		int 	`json:"udpPort"`
//	Ip 			string 	`json:"ip"`
//}

var PlayerPool map[int32]*myproto.Player //玩家 状态池
func PlayersNew()*Players {
	mylog.Info("new Players instance")
	players := new(Players)
	PlayerPool = make(map[int32]*myproto.Player)
	return players
}
func  (players *Players)addPlayerPool(id int32)(existPlayer myproto.Player,err error){
	mylog.Info("addPlayerPool :",id)
	hasPlayer,empty  := players.GetById(id)
	if !empty{
		mylog.Notice("this player id has exist pool",id)
		if hasPlayer.Status == PLAYER_STATUS_ONLINE {
			errMsg := "hasPlayer.Status = PLAYER_STATUS_ONLINE "
			mylog.Error(errMsg)
			err = errors.New(errMsg)
			return *hasPlayer,err
		}else{
			players.upPlayerStatus(id, PLAYER_STATUS_ONLINE)
			return *hasPlayer,nil
		}
	}else{
		mylog.Info("new player add")
		player := myproto.Player{
			Id:       id,
			AddTime:  int32(zlib.GetNowTimeSecondToInt()),
			Nickname: "",
			Status:   PLAYER_STATUS_ONLINE,
		}
		PlayerPool[id] = &player
		return player,nil
	}
	return existPlayer,nil
}
func  (players *Players)GetById(playerId int32)(player *myproto.Player,empty bool){
	myPlayer ,ok := PlayerPool[playerId]
	if ok {
		return myPlayer ,false
	}else{
		return player,true
	}
}

//func  (players *Players)getPlayerStatus(requestPlayerStatus RequestPlayerStatus,wsConn *WsConn){
//	player,_ := players.getById(requestPlayerStatus.PlayerId)
//	responsePlayerStatus := ResponsePlayerStatus{
//		Id :player.Id,
//		Nickname: player.Nickname,
//		Status : player.Status,
//		AddTime: player.AddTime,
//		UpTime 	: player.UpTime,
//		RoomId 	: player.RoomId,
//	}
//	responsePlayerStatusStr,_ := json.Marshal(responsePlayerStatus)
//	mynetWay.SendMsgByUid(wsConn.PlayerId,"pushPlayerStatus",string(responsePlayerStatusStr))
//}
func  (players *Players)delById(playerId int32){
	delete(PlayerPool,playerId)
}
func   (players *Players)upPlayerStatus(id int32,status int32){
	player := PlayerPool[id]

	mylog.Info("upPlayerStatus" , " old : ",player.Status," new:",status)

	player.Status = status
	player.UpTime = int32(zlib.GetNowTimeSecondToInt())

}

func   (players *Players)UpPlayerRoomId(playerId int32,roomId string){
	player := PlayerPool[playerId]

	mylog.Info("upPlayerRoomId" , " old : ",player.RoomId," new:",roomId)

	player.RoomId = roomId
	player.UpTime = int32 (zlib.GetNowTimeSecondToInt())

}
