package env

import "os"

// Load a *required* string environment variable.
// This will panic if the variable is not set.
func loadRequired(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic("Environment variable " + name + " not set.")
	}
	return value
}

// Load am *optional* string environment variable.
// This will return an empty string if the variable is not set.
func loadOptional(name string) string {
	return os.Getenv(name)
}

// The path under which the history files are stored, from the environment variable.
var StaticPath = loadRequired("STATIC_PATH")

// The SensorThings API base URL.
var SensorThingsBaseUrl = loadRequired("SENSORTHINGS_URL")

// The URL to the observation MQTT broker from the environment variable.
var SensorThingsObservationMqttUrl = loadRequired("SENSORTHINGS_MQTT_URL")

// The url to the prediction service MQTT broker.
var PredictionServiceMqttUrl = loadRequired("PREDICTION_SERVICE_MQTT_URL")

// The username to use for the prediction service MQTT broker.
var PredictionServiceMqttUsername = loadOptional("PREDICTION_SERVICE_MQTT_USERNAME")

// The password to use for the prediction service MQTT broker.
var PredictionServiceMqttPassword = loadOptional("PREDICTION_SERVICE_MQTT_PASSWORD")

// The url to the predictor MQTT broker.
var PredictorMqttUrl = loadRequired("PREDICTOR_MQTT_URL")

// The username to use for the predictor MQTT broker.
var PredictorMqttUsername = loadOptional("PREDICTOR_MQTT_USERNAME")

// The password to use for the predictor MQTT broker.
var PredictorMqttPassword = loadOptional("PREDICTOR_MQTT_PASSWORD")
