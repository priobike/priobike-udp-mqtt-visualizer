package monitor

import (
	"io/ioutil"
	"mqttudpvisualizer/env"
	"mqttudpvisualizer/log"
	"mqttudpvisualizer/observations"
	"mqttudpvisualizer/phases"
	"mqttudpvisualizer/predictions"
	"mqttudpvisualizer/things"
	"sort"
	"time"

	geojson "github.com/paulmach/go.geojson"
)

// Write geojson data that can be used to visualize the predictions.
// The geojson file is written to the static directory.
func WriteGeoJSONMap() {
	log.Info.Println("Writing geojson map.")
	// Write the geojson to the file.
	locationFeatureCollection := geojson.NewFeatureCollection() // Locations of traffic lights.

	// Get the thing names and order them alphabetically.
	thingsOrdered := []string{}
	things.Things.Range(func(key, value interface{}) bool {
		thingName := key.(string)
		thingsOrdered = append(thingsOrdered, thingName)
		return true
	})
	sort.Strings(thingsOrdered)

	for _, thingName := range thingsOrdered {
		thingAny, ok := things.Things.Load(thingName)
		if !ok {
			log.Error.Println("Thing not found:", thingName)
			continue
		}
		thing := thingAny.(things.Thing)
		lane, err := thing.Lane()
		if err != nil {
			// Some things may not have lanes.
			continue
		}
		coordinate := lane[0]
		lat, lng := coordinate[1], coordinate[0]

		// Check if there is a prediction for this thing.
		pPrediction, pPredictionOk := predictions.GetCurrentPredictorPrediction(thingName)
		psPrediction, psPredictionOk := predictions.GetCurrentPredictionServicePrediction(thingName)
		currentPrimarySignal, currentPrimarySignalOk := observations.GetCurrentPrimarySignal(thingName)

		// Build the properties.
		properties := make(map[string]interface{})
		properties["n"] = thing.Name
		properties["l"] = thing.Properties.LaneType

		// Add the prediction properties.
		if pPredictionOk {
			properties["p"] = pPrediction.GetCurrentPredictedState()
		} else {
			properties["p"] = phases.Unknown
		}
		if psPredictionOk {
			properties["ps"] = psPrediction.GetCurrentPredictedState()
		} else {
			properties["ps"] = phases.Unknown
		}
		if currentPrimarySignalOk {
			properties["a"] = currentPrimarySignal.Result
			properties["at"] = int(time.Since(currentPrimarySignal.PhenomenonTime).Seconds())
		} else {
			properties["a"] = phases.Unknown
			properties["at"] = -1
		}

		// Make a point feature.
		location := geojson.NewPointFeature([]float64{lng, lat})
		location.Properties = properties
		locationFeatureCollection.AddFeature(location)
	}

	locationsGeoJson, err := locationFeatureCollection.MarshalJSON()
	if err != nil {
		log.Error.Println("Error marshalling geojson:", err)
		return
	}
	ioutil.WriteFile(env.StaticPath+"/map.geojson", locationsGeoJson, 0644)
}

func UpdateGeoJSONMapPeriodically() {
	for {
		time.Sleep(1 * time.Second)
		WriteGeoJSONMap()
	}
}
