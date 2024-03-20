package predictions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"mqttudpvisualizer/env"
	"mqttudpvisualizer/log"
	"mqttudpvisualizer/phases"
	"mqttudpvisualizer/things"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var predictionServicePredictions = &sync.Map{}

type PredictionServicePrediction struct {
	GreenTimeThreshold float64   `json:"greentimeThreshold"`
	PredictionQuality  float64   `json:"predictionQuality"`
	SignalGroupId      string    `json:"signalGroupId"`
	StartTime          time.Time `json:"startTime"`
	Value              []byte    `json:"value"`
}

// Custom unmarshal function that removes Remove [UTC] from the startTime.
func (p *PredictionServicePrediction) UnmarshalJSON(data []byte) error {
	type alias struct {
		GreenTimeThreshold float64 `json:"greentimeThreshold"`
		PredictionQuality  float64 `json:"predictionQuality"`
		SignalGroupId      string  `json:"signalGroupId"`
		StartTime          string  `json:"startTime"`
		Value              []byte  `json:"value"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	p.GreenTimeThreshold = a.GreenTimeThreshold
	p.PredictionQuality = a.PredictionQuality
	p.SignalGroupId = a.SignalGroupId
	p.Value = a.Value

	// Parse the startTime.
	// Remove [UTC] from the startTime.
	a.StartTime = a.StartTime[:len(a.StartTime)-len("[UTC]")]
	startTime, err := time.Parse(time.RFC3339, a.StartTime)
	if err != nil {
		return err
	}
	p.StartTime = startTime
	return nil
}

// Get the current predicted state.
func (p PredictionServicePrediction) GetCurrentPredictedState() byte {
	if p.GreenTimeThreshold == -1 {
		return phases.Unknown
	}
	if p.PredictionQuality == -1 {
		return phases.Unknown
	}
	if len(p.Value) == 0 {
		return phases.Unknown
	}
	timesince := int(time.Since(p.StartTime).Seconds())
	if timesince >= len(p.Value) {
		return phases.Unknown
	}
	greenNow := p.Value[timesince] >= byte(p.GreenTimeThreshold)
	if greenNow {
		return phases.Green
	} else {
		return phases.Red
	}
}

// Get the most recent prediction service prediction for this thing.
func GetCurrentPredictionServicePrediction(thingName string) (PredictionServicePrediction, bool) {
	prediction, ok := predictionServicePredictions.Load(thingName)
	if !ok {
		return PredictionServicePrediction{}, false
	}
	return prediction.(PredictionServicePrediction), true
}

// The number of processed messages, for logging purposes.
var PredictionServicePredictionsReceived uint64 = 0

// Check out the number of received messages periodically.
func CheckReceivedPredictionServiceMessagesPeriodically() {
	for {
		receivedNow := PredictionServicePredictionsReceived
		time.Sleep(60 * time.Second)
		receivedThen := PredictionServicePredictionsReceived
		dReceived := receivedThen - receivedNow
		// Panic if the number of received messages is too low.
		if dReceived == 0 {
			panic("No prediction service messages received in the last 60 seconds")
		}
		log.Info.Printf("Received %d prediction service predictions in the last 60 seconds.", dReceived)
	}
}

func processPredictionServiceMessage(msg mqtt.Message) {
	atomic.AddUint64(&PredictionServicePredictionsReceived, 1)

	// Add the observation to the correct map.
	topic := msg.Topic()
	thingName := topic[len("hamburg/"):]

	var prediction PredictionServicePrediction
	if err := json.Unmarshal(msg.Payload(), &prediction); err != nil {
		log.Error.Println("Error unmarshalling prediction service prediction:", err)
		return
	}

	predictionServicePredictions.Store(thingName, prediction)
}

// Run an MQTT client that listens for prediction service predictions.
func ConnectPredictionServiceMqttListener() {
	topics := []string{}
	things.Things.Range(func(key, value interface{}) bool {
		thingName := key.(string)
		topics = append(topics, fmt.Sprintf("hamburg/%s", thingName))
		return true
	})

	// Create a new client for every 1000 subscriptions.
	// Otherwise messages will queue up after some time, since the client
	// is not parallelized enough. This is a workaround for the issue.
	// Bonus points: this also reduces CPU usage significantly.
	var client mqtt.Client
	var wg sync.WaitGroup
	for i, topic := range topics {
		if (i % 1000) == 0 {
			wg.Wait()
			opts := mqtt.NewClientOptions()
			opts.AddBroker(env.PredictionServiceMqttUrl)
			opts.SetConnectTimeout(10 * time.Second)
			opts.SetConnectRetry(true)
			opts.SetConnectRetryInterval(5 * time.Second)
			opts.SetAutoReconnect(true)
			opts.SetKeepAlive(60 * time.Second)
			opts.SetPingTimeout(10 * time.Second)
			opts.SetOnConnectHandler(func(client mqtt.Client) {
				log.Info.Printf(
					"Connected to prediction service mqtt broker: %s",
					env.PredictionServiceMqttUrl,
				)
			})
			opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
				log.Warning.Println("Connection to prediction service mqtt broker lost:", err)
			})
			randSource := rand.NewSource(time.Now().UnixNano())
			random := rand.New(randSource)
			clientID := fmt.Sprintf("priobike-udpmqttvisualizer-%d", random.Int())
			opts.SetClientID(clientID)
			opts.SetOrderMatters(false)
			opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
				log.Warning.Println("Received unexpected message on topic:", msg.Topic())
			})
			if env.PredictionServiceMqttUsername != "" {
				opts.SetUsername(env.PredictionServiceMqttUsername)
			}
			if env.PredictionServiceMqttPassword != "" {
				opts.SetPassword(env.PredictionServiceMqttPassword)
			}

			client = mqtt.NewClient(opts)
			if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
				panic(conn.Error())
			}
		}

		wg.Add(1)
		go func(topic string) {
			defer wg.Done()

			// Subscribe to the datastream.
			if token := client.Subscribe(topic, 1 /* QoS */, func(client mqtt.Client, msg mqtt.Message) {
				// Process the message asynchronously to avoid blocking the mqtt client.
				go processPredictionServiceMessage(msg)
			}); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		}(topic)
	}

	log.Info.Println("Subscribed to all prediction service prediction topics.")
}
