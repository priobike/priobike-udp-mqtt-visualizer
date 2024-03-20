package main

import (
	"mqttudpvisualizer/monitor"
	"mqttudpvisualizer/observations"
	"mqttudpvisualizer/predictions"
	"mqttudpvisualizer/things"
)

func main() {
	things.SyncThings()

	observations.PrefetchMostRecentObservations()
	observations.ConnectObservationListener()
	go observations.CheckReceivedMessagesPeriodically()

	predictions.ConnectPredictorMqttListener()
	go predictions.CheckReceivedPredictorMessagesPeriodically()
	predictions.ConnectPredictionServiceMqttListener()
	go predictions.CheckReceivedPredictionServiceMessagesPeriodically()

	monitor.WriteGeoJSONMap()
	go monitor.UpdateGeoJSONMapPeriodically()

	// Wait forever.
	select {}
}
