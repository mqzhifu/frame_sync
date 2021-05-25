package netway

import (
	"context"
	"zlib"
)

type Metrics struct {
	input 	chan MetricsChanMsg
	totalMetrics 	TotalMetrics
	roomSyncMetrics map[string]RoomSyncMetrics
	Pool 	map[string]int
}


type MetricsChanMsg struct {
	Key 	string
	Value 	int
	Opt 	int	//1累加2加加3累减4减减
}

type TotalMetrics struct{
	InputNum		int `json:"inputNum"`
	InputSize		int `json:"inputSize"`
	OutputNum		int `json:"outputNum"`
	OutputSize		int `json:"outputSize"`
	FDNum			int `json:"fdNum"`
	FDCreateFail	int `json:"fdCreateFail"`
	RoomNum 		int `json:"roomNum"`
}

type RoomSyncMetrics struct{
	InputNum	int `json:"inputNum"`
	InputSize	int `json:"inputSize"`
	OutputNum	int `json:"outputNum"`
	OutputSize	int `json:"outputSize"`
}

func MetricsNew()*Metrics{
	metrics := new (Metrics)
	metrics.input = make(chan MetricsChanMsg ,100)
	metrics.totalMetrics = TotalMetrics{}
	metrics.roomSyncMetrics = make(map[string]RoomSyncMetrics)
	metrics.Pool = make(map[string]int)
	return metrics
	//myMetrics = zlib.NewMetrics()
}
func  (metrics *Metrics)fastLog(key string,opt int ,value int ){
	metricsChanMsg := MetricsChanMsg{
		Key :key,
		Opt: opt,
		Value: value,
	}
	metrics.input <- metricsChanMsg
}
func  (metrics *Metrics)start(ctx context.Context){
	ctxHasDone := 0
	for{
		select {
			case metricsChanMsg := <- metrics.input:
				metrics.processMsg(metricsChanMsg)
			case <- ctx.Done():
				ctxHasDone = 1
				mylog.Warning("checkRoomTimeoutLoop done.")
		}
		if ctxHasDone == 1{
			goto end
		}
	}
	end:
		zlib.MyPrint("end:checkRoomTimeoutLoop done.")

}

func (metrics *Metrics)processMsg(metricsChanMsg MetricsChanMsg){
	if metricsChanMsg.Opt == 1{
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " add " + strconv.Itoa( metricsChanMsg.Value))
		metrics.Pool[metricsChanMsg.Key] += metricsChanMsg.Value
	}else if metricsChanMsg.Opt == 2 {
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " ++ ")
		metrics.Pool[metricsChanMsg.Key]++
	}else if metricsChanMsg.Opt == 3{
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " add " + strconv.Itoa( metricsChanMsg.Value))
		metrics.Pool[metricsChanMsg.Key] -= metricsChanMsg.Value
	}else if metricsChanMsg.Opt == 4{
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " -- ")
		metrics.Pool[metricsChanMsg.Key]--
	}
}
