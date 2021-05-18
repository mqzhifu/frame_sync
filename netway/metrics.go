package netway

import (
	"context"
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
	Opt 	int
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

func  (metrics *Metrics)start(ctx context.Context){
	for{
		select {
		case metricsChanMsg := <- metrics.input:
			metrics.processMsg(metricsChanMsg)
		case <- ctx.Done():
			mylog.Warning("checkRoomTimeoutLoop done.")
			return
		default:
			//time.Sleep(1 * time.Second)
		}
	}
}

func (metrics *Metrics)processMsg(metricsChanMsg MetricsChanMsg){
	if metricsChanMsg.Opt == 1{
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " add " + strconv.Itoa( metricsChanMsg.Value))
		metrics.Pool[metricsChanMsg.Key] += metricsChanMsg.Value
	}else if metricsChanMsg.Opt == 2{
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " ++ ")
		metrics.Pool[metricsChanMsg.Key]++
	}else if metricsChanMsg.Opt == 4{
		//mylog.Debug("metrics :"+metricsChanMsg.Key + " -- ")
		metrics.Pool[metricsChanMsg.Key]--
	}
}
