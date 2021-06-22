package netway

import (
	"context"
)

type Metrics struct {
	input 	chan MetricsChanMsg
	totalMetrics 	TotalMetrics
	roomSyncMetrics map[string]RoomSyncMetrics
	Pool 	map[string]int
	Close chan int
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

func NewMetrics()*Metrics{
	mylog.Info("NewMetrics instance:")
	metrics := new (Metrics)
	metrics.input = make(chan MetricsChanMsg ,100)
	metrics.totalMetrics = TotalMetrics{}
	metrics.roomSyncMetrics = make(map[string]RoomSyncMetrics)
	metrics.Pool = make(map[string]int)
	metrics.Close = make(chan int)
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
func (metrics *Metrics)Shutdown(){
	mylog.Alert("shutdown metrics")
	metrics.Close <- 1
}

func  (metrics *Metrics)start(ctx context.Context){
	defer func(ctx context.Context ) {
		if err := recover(); err != nil {
			myNetWay.RecoverGoRoutine(metrics.start,ctx,err)
		}
	}(ctx)

	mylog.Alert("metrics start:")
	ctxHasDone := 0
	for{
		select {
			case metricsChanMsg := <- metrics.input:
				metrics.processMsg(metricsChanMsg)
			case <- metrics.Close:
				ctxHasDone = 1
		}
		if ctxHasDone == 1{
			goto end
		}
	}
	end:
		mylog.Alert(CTX_DONE_PRE+" metrics.")
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
