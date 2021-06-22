package netway

import (
	"context"
	"frame_sync/myproto"
	"strconv"
	"time"
	"zlib"
)

type Match struct {
	Option MatchOption
	RecoverTimes int
	Close 	chan int
}

type MatchOption struct {
	RoomPeople	int32
	RoomReadyTimeout int32
	MatchSuccessChan chan *Room
}

type PlayerSign struct {
	AddTime 	int32
	PlayerId	int32
}

var signPlayerPool []PlayerSign
func NewMatch(matchOption MatchOption)*Match {
	mylog.Info("NewMatch instance")
	match  := new(Match)
	if matchOption.RoomPeople <= 1{
		zlib.PanicPrint("roomPeople <= 1")
	}
	match.Option = matchOption
	match.Close = make(chan int)
	return match
}

func (match *Match)getOneSignPlayerById(playerId int32 ) (playerSign PlayerSign,empty bool){
	for _,v := range signPlayerPool {
		if v.PlayerId == playerId {
			return v,false
		}
	}
	return playerSign,true
}

func (match *Match) addOnePlayer(requestPlayerMatchSign myproto.RequestPlayerMatchSign,conn *Conn){
	playerId := conn.PlayerId
	player ,empty := myPlayerManager.GetById(requestPlayerMatchSign.PlayerId)
	if empty{
		msg := "playerMatchSign getPlayById is empty~id:"+strconv.Itoa(int(requestPlayerMatchSign.PlayerId))
		match.matchSignErrAndSend(msg,conn)
		return
	}

	if player.Status != PLAYER_STATUS_ONLINE{
		msg := "playerMatchSign status != online , status="+ strconv.Itoa(int(player.Status))
		match.matchSignErrAndSend(msg,conn)
		return
	}

	if player.RoomId != "" {
		msg := "playerMatchSign player.RoomId != '' , roomId=" +   player.RoomId
		match.matchSignErrAndSend(msg,conn)
		return
	}
	//err := myMatch.addOnePlayer(requestPlayerMatchSign.PlayerId)
	//if err != nil{
	//	mylog.Error("playerReady",err.Error())
	//	return
	//}
	_,empty = match.getOneSignPlayerById(playerId)
	if !empty{
		msg :="match sign addOnePlayer : player has exist"+ strconv.Itoa(int(playerId))
		match.matchSignErrAndSend(msg,conn)
		return
	}
	newPlayerSign := PlayerSign{PlayerId: playerId,AddTime: int32(zlib.GetNowTimeSecondToInt())}
	signPlayerPool = append(signPlayerPool,newPlayerSign)
	return
}
//玩家报名时，可能因为BUG，造成一些系统级的错误，如：丢失玩家状态等
//出现这种S端异常的情况，除了报错还要通知一下C端
func (match *Match)matchSignErrAndSend(msg string,conn *Conn){
	mylog.Error(msg)
	playerMatchSignFailed := myproto.ResponsePlayerMatchSignFailed{
		PlayerId: conn.PlayerId,
		Msg: msg,
	}
	myNetWay.SendMsgCompressByUid(conn.PlayerId,"playerMatchSignFailed",&playerMatchSignFailed)
}

func (match *Match) delOnePlayer(requestCancelSign myproto.RequestPlayerMatchSignCancel,conn *Conn){
	match.realDelOnePlayer(requestCancelSign.PlayerId)
}
func (match *Match) realDelOnePlayer(playerId int32){
	//playerId := requestCancelSign.PlayerId
	mylog.Info("cancel : delOnePlayer ",playerId)
	for k,v:=range signPlayerPool {
		if v.PlayerId == playerId{
			if len(signPlayerPool) == 1{
				signPlayerPool = []PlayerSign{}
			}else{
				signPlayerPool = append(signPlayerPool[:k], signPlayerPool[k+1:]...)
			}
			return
		}
	}
	mylog.Warning("no match playerId",playerId)
}
func (match *Match)Shutdown(){
	mylog.Alert("shutdown match")
	match.Close <- 1
}
func (match *Match) Start( ctx context.Context ){
	defer func(ctx context.Context ) {
		if err := recover(); err != nil {
			myNetWay.RecoverGoRoutine(match.Start,ctx,err)
		}
	}(ctx)

	matchSuccessChan := match.Option.MatchSuccessChan
	mylog.Alert("matchingPlayerCreateRoom:start")
	for{
		select {
			case   <- match.Close:
				goto end
			default:
				//计算：当前池子里的人数，是否满足可以开启一局游戏
				if int32(len(signPlayerPool)) < match.Option.RoomPeople{
					//不满足即睡眠等待500毫秒
					time.Sleep(time.Millisecond * 500)
					break
				}
				//创建一个新的房间，用于装载玩家及同步数据等
				newRoom := NewRoom()
				//timeout := int32(zlib.GetNowTimeSecondToInt()) + mynetWay.Option.RoomTimeout
				//newRoom.Timeout = int32(timeout)
				readyTimeout := int32(zlib.GetNowTimeSecondToInt()) + match.Option.RoomReadyTimeout
				newRoom.ReadyTimeout = readyTimeout
				for i:=0;i < len(signPlayerPool);i++{
					player,empty := myPlayerManager.GetById(signPlayerPool[i].PlayerId)
					if empty{
						mylog.Error("matching Players.getById empty , ", signPlayerPool[i].PlayerId)
					}
					if player.Status != PLAYER_STATUS_ONLINE{
						mylog.Error("matching Players.status != online , ", signPlayerPool[i].PlayerId)
					}
					if player.RoomId != ""{
						mylog.Error("matching Players.roomId != '' , ", signPlayerPool[i].PlayerId)
					}
					newRoom.AddPlayer(player)
				}
				//删除上面匹配成功的玩家
				signPlayerPool = append(signPlayerPool[match.Option.RoomPeople:])
				mylog.Info("create a room :",newRoom)
				//将该房间添加到管道中
				matchSuccessChan <- newRoom
				//time.Sleep(time.Millisecond * 100)
		}
	}
end:
	mylog.Alert(CTX_DONE_PRE+"matchingPlayerCreateRoom close")
}

