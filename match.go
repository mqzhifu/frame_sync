package main

import (
	"context"
	"time"
	"zlib"
)

type Match struct {
	Option MatchOption
}

type MatchOption struct {
	RoomPeople	int
}

type PlayerSign struct {
	AddTime 	int
	PlayerId	int
}

var signPlayerPool []PlayerSign
func NewMatch(matchOption MatchOption)*Match{
	match  := new(Match)
	match.Option = matchOption
	return match
}

func (match *Match) addOnePlayer(playerId int){
	playerSign := PlayerSign{PlayerId: playerId,AddTime: zlib.GetNowTimeSecondToInt()}
	signPlayerPool = append(signPlayerPool,playerSign)
}

func (match *Match) delOnePlayer(playerId int){

	for k,v:=range signPlayerPool{
		if v.PlayerId == playerId{
			if len(signPlayerPool ) == 1{
				signPlayerPool = []PlayerSign{}
			}else{
				signPlayerPool = append(signPlayerPool[:k], signPlayerPool[k+1:]...)
			}
		}
	}
}

func (match *Match) matchingPlayerCreateRoom(ctx context.Context){
	mylog.Info("matchingPlayerCreateRoom:start")
	for{
		select {
		case   <-ctx.Done():
			//netWay.Option.Mylog.Warning("matchingPlayerCreateRoom close")
			goto end
		default:
			//zlib.MyPrint(len(signPlayerPool))
			//mylog.Info("matching:",len(signPlayerPool),match.Option.RoomPeople)
			if len(signPlayerPool) >= match.Option.RoomPeople{
				newRoom := NewRoom()
				for i:=0;i < len(signPlayerPool);i++{
					player := Player{
						Id: signPlayerPool[i].PlayerId,
					}
					newRoom.AddPlayer(player)
				}
				//zlib.MyPrint(newRoom)
				//zlib.MyPrint(signPlayerPool)
				signPlayerPool = append(signPlayerPool[match.Option.RoomPeople:])
				//zlib.ExitPrint(signPlayerPool)
				mylog.Info("create a room :",newRoom)
				mySync.addPoolElement(newRoom)
				mySync.start(newRoom.Id)
			}
			time.Sleep(time.Second * 1)
			//mySleepSecond(1,"matching player")
			//mySleepSecond(1,"")
		}
	}
end:
	mylog.Info("matchingPlayerCreateRoom close")
}

