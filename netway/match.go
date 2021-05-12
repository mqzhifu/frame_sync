package netway

import (
	"context"
	"errors"
	"time"
	"zlib"
)

type Match struct {
	Option MatchOption
}

type MatchOption struct {
	RoomPeople	int32
}

type PlayerSign struct {
	AddTime 	int32
	PlayerId	int32
}

var signPlayerPool []PlayerSign
func NewMatch(matchOption MatchOption)*Match {
	mylog.Info("NewMatch instance")
	match  := new(Match)
	match.Option = matchOption
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

func (match *Match) addOnePlayer(playerId int32)error{
	_,empty := match.getOneSignPlayerById(playerId)
	if !empty{
		return errors.New("match sign addOnePlayer : player has exist")
	}
	newPlayerSign := PlayerSign{PlayerId: playerId,AddTime: int32(zlib.GetNowTimeSecondToInt())}
	signPlayerPool = append(signPlayerPool,newPlayerSign)
	return nil
}

func (match *Match) delOnePlayer(playerId int32){
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

func (match *Match) matchingPlayerCreateRoom(ctx context.Context,matchSuccessChan chan *Room){
	mylog.Info("matchingPlayerCreateRoom:start")
	for{
		select {
		case   <-ctx.Done():
			//netWay.Option.Mylog.Warning("matchingPlayerCreateRoom close")
			goto end
		default:
			//zlib.MyPrint(len(signPlayerPool))
			//mylog.Info("matching:",len(signPlayerPool),match.Option.RoomPeople)
			if int32(len(signPlayerPool)) >= match.Option.RoomPeople{
				newRoom := NewRoom()
				timeout := int32(zlib.GetNowTimeSecondToInt()) + mynetWay.Option.RoomTimeout
				newRoom.Timeout = int32(timeout)
				readyTimeout := int32(zlib.GetNowTimeSecondToInt()) + mynetWay.Option.RoomReadyTimeout
				newRoom.ReadyTimeout = readyTimeout
				for i:=0;i < len(signPlayerPool);i++{
					player,empty := mynetWay.Players.GetById(signPlayerPool[i].PlayerId)
					if empty{
						mylog.Error("match Players.getById empty , ", signPlayerPool[i].PlayerId)
					}
					player.RoomId = newRoom.Id
					newRoom.AddPlayer(player)
					newRoom.PlayersReadyList[player.Id] = 0
				}
				//删除上面匹配成功的玩家
				signPlayerPool = append(signPlayerPool[match.Option.RoomPeople:])
				mylog.Info("create a room :",newRoom)
				//将该房间添加到容器中
				matchSuccessChan <- newRoom


			}
			time.Sleep(time.Second * 1)
			//mySleepSecond(1,"matching player")
		}
	}
end:
	mylog.Info("matchingPlayerCreateRoom close")
}

