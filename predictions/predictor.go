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

var predictorPredictions = &sync.Map{}

type PredictorPrediction struct {
	ThingName     string    `json:"thingName"`
	Now           []byte    `json:"now"`         // Will be serialized to a base64 string.
	NowQuality    []byte    `json:"nowQuality"`  // Will be serialized to a base64 string.
	Then          []byte    `json:"then"`        // Will be serialized to a base64 string.
	ThenQuality   []byte    `json:"thenQuality"` // Will be serialized to a base64 string.
	ReferenceTime time.Time `json:"referenceTime"`
	ProgramId     *byte     `json:"programId"`
}

// Calculate the average quality of a predictor prediction.
func (p PredictorPrediction) AverageQuality() float64 {
	qualitySum := 0
	qualityLength := 0
	for _, quality := range p.NowQuality {
		qualitySum += int(quality)
		qualityLength++
	}
	for _, quality := range p.ThenQuality {
		qualitySum += int(quality)
		qualityLength++
	}
	if qualityLength > 0 {
		return float64(qualitySum) / float64(qualityLength)
	} else {
		return 0
	}
}

// Get the current predicted state.
func (p PredictorPrediction) GetCurrentPredictedState() byte {
	// The prediction is split into two parts: "now" and "then".
	// "now" is the predicted behavior within the current cycle, which can deviate from the average behavior.
	// "then" is the predicted behavior after the cycle, which is the average behavior.
	lenNow := len(p.Now)
	lenThen := len(p.Then)
	if lenNow == 0 && lenThen == 0 {
		return phases.Unknown
	}
	var timesince = int(time.Since(p.ReferenceTime).Seconds())
	if timesince == -1 {
		// Small deviation is ok, set to 0
		timesince = 0
	}
	if timesince < 0 {
		log.Warning.Println("Encountered negative timesince in predictor prediction:", timesince)
		return phases.Unknown
	}
	if timesince < len(p.Now) {
		i := timesince % lenNow
		return p.Now[i]
	} else {
		i := (timesince - lenNow) % lenThen
		return p.Then[i]
	}
}

// Get the most recent predictor prediction for this thing.
func GetCurrentPredictorPrediction(thingName string) (PredictorPrediction, bool) {
	prediction, ok := predictorPredictions.Load(thingName)
	if !ok {
		return PredictorPrediction{}, false
	}
	return prediction.(PredictorPrediction), true
}

// The number of processed messages, for logging purposes.
var PredictorPredictionsReceived uint64 = 0

// Check out the number of received messages periodically.
func CheckReceivedPredictorMessagesPeriodically() {
	for {
		receivedNow := PredictorPredictionsReceived
		time.Sleep(60 * time.Second)
		receivedThen := PredictorPredictionsReceived
		dReceived := receivedThen - receivedNow
		// Panic if the number of received messages is too low.
		if dReceived == 0 {
			panic("No predictor messages received in the last 60 seconds")
		}
		log.Info.Printf("Received %d predictor predictions in the last 60 seconds.", dReceived)
	}
}

func processPredictorMessage(msg mqtt.Message) {
	atomic.AddUint64(&PredictorPredictionsReceived, 1)

	// Add the observation to the correct map.
	topic := msg.Topic()
	thingName := topic[len("hamburg/"):]

	var prediction PredictorPrediction
	if err := json.Unmarshal(msg.Payload(), &prediction); err != nil {
		log.Error.Println("Error unmarshalling predictor prediction:", err)
		return
	}

	predictorPredictions.Store(thingName, prediction)
}

// Run an MQTT client that listens for predictor predictions.
func ConnectPredictorMqttListener() {
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
			opts.AddBroker(env.PredictorMqttUrl)
			opts.SetConnectTimeout(10 * time.Second)
			opts.SetConnectRetry(true)
			opts.SetConnectRetryInterval(5 * time.Second)
			opts.SetAutoReconnect(true)
			opts.SetKeepAlive(60 * time.Second)
			opts.SetPingTimeout(10 * time.Second)
			opts.SetOnConnectHandler(func(client mqtt.Client) {
				log.Info.Printf(
					"Connected to predictor mqtt broker: %s",
					env.PredictorMqttUrl,
				)
			})
			opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
				log.Warning.Println("Connection to predictor mqtt broker lost:", err)
			})
			randSource := rand.NewSource(time.Now().UnixNano())
			random := rand.New(randSource)
			clientID := fmt.Sprintf("priobike-udpmqttvisualizer-%d", random.Int())
			opts.SetClientID(clientID)
			opts.SetOrderMatters(false)
			opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
				log.Warning.Println("Received unexpected message on topic:", msg.Topic())
			})
			if env.PredictorMqttUsername != "" {
				opts.SetUsername(env.PredictorMqttUsername)
			}
			if env.PredictorMqttPassword != "" {
				opts.SetPassword(env.PredictorMqttPassword)
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
				go processPredictorMessage(msg)
			}); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		}(topic)
	}

	log.Info.Println("Subscribed to all predictor prediction topics.")
}
