package observations

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"mqttudpvisualizer/env"
	"mqttudpvisualizer/log"
	"mqttudpvisualizer/things"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// The QoS for observations. We use QoS 1 (at least once).
// We also don't use QoS 2 since the messages could be delayed if the
// broker is overloaded or the upstream connection is slow. Note
// that this implies that we might receive the same observation twice.
const observationQoS = 1

// Received messages by their topic.
var ObservationsReceivedByTopic = make(map[string]uint64)

// The lock for the map.
var ObservationsReceivedByTopicLock = &sync.RWMutex{}

// The number of processed messages, for logging purposes.
var ObservationsReceived uint64 = 0
var ObservationsDiscarded uint64 = 0
var ObservationsProcessed uint64 = 0

// Check out the number of received messages periodically.
func CheckReceivedMessagesPeriodically() {
	for {
		receivedNow := ObservationsReceived
		canceledNow := ObservationsDiscarded
		processedNow := ObservationsProcessed
		time.Sleep(60 * time.Second)
		receivedThen := ObservationsReceived
		canceledThen := ObservationsDiscarded
		processedThen := ObservationsProcessed
		dReceived := receivedThen - receivedNow
		dCanceled := canceledThen - canceledNow
		dProcessed := processedThen - processedNow
		// Panic if the number of received messages is too low.
		if dReceived == 0 {
			panic("No messages received in the last 60 seconds")
		}
		log.Info.Printf("Received %d observations in the last 60 seconds. (%d processed, %d canceled)", dReceived, dProcessed, dCanceled)
		ObservationsReceivedByTopicLock.RLock()
		for dsType, count := range ObservationsReceivedByTopic {
			log.Info.Printf("  - Received %d observations for `%s`.", count, dsType)
		}
		ObservationsReceivedByTopicLock.RUnlock()
	}
}

// Process a message.
func processMessage(msg mqtt.Message) {
	atomic.AddUint64(&ObservationsReceived, 1)

	// Add the observation to the correct map.
	topic := msg.Topic()

	// Check if the topic should be processed.
	dsType, ok := things.DatastreamMqttTopics.Load(topic)
	if !ok {
		atomic.AddUint64(&ObservationsDiscarded, 1)
		return
	}

	// Increment the number of received messages.
	ObservationsReceivedByTopicLock.Lock()
	ObservationsReceivedByTopic[dsType.(string)]++
	ObservationsReceivedByTopicLock.Unlock()

	var observation Observation
	if err := json.Unmarshal(msg.Payload(), &observation); err != nil {
		atomic.AddUint64(&ObservationsDiscarded, 1)
		return
	}

	err := validateObservation(observation, dsType.(string))
	if err != nil {
		atomic.AddUint64(&ObservationsDiscarded, 1)
		log.Warning.Printf("Invalid observation: %s", err)
		return
	}

	switch dsType {
	case "primary_signal":
		thingName, ok := things.PrimarySignalDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&ObservationsDiscarded, 1)
			return
		}
		currentPrimarySignal.Store(thingName, &observation)
	case "signal_program":
		thingName, ok := things.SignalProgramDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&ObservationsDiscarded, 1)
			return
		}
		currentSignalProgram.Store(thingName, &observation)
	case "detector_car":
		thingName, ok := things.CarDetectorDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&ObservationsDiscarded, 1)
			return
		}
		currentCarDetector.Store(thingName, &observation)
	case "detector_bike":
		thingName, ok := things.BikeDetectorDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&ObservationsDiscarded, 1)
			return
		}
		currentBikeDetector.Store(thingName, &observation)
	case "cycle_second":
		thingName, ok := things.CycleSecondDatastreams.Load(topic)
		if !ok {
			atomic.AddUint64(&ObservationsDiscarded, 1)
			return
		}
		currentCycleSecond.Store(thingName, &observation)
	}

	atomic.AddUint64(&ObservationsProcessed, 1)
}

// Listen for new observations via mqtt.
func ConnectObservationListener() {
	topics := []string{}
	things.DatastreamMqttTopics.Range(func(topic, _ interface{}) bool {
		topics = append(topics, topic.(string))
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
			opts.AddBroker(env.SensorThingsObservationMqttUrl)
			opts.SetConnectTimeout(10 * time.Second)
			opts.SetConnectRetry(true)
			opts.SetConnectRetryInterval(5 * time.Second)
			opts.SetAutoReconnect(true)
			opts.SetKeepAlive(60 * time.Second)
			opts.SetPingTimeout(10 * time.Second)
			opts.SetOnConnectHandler(func(client mqtt.Client) {
				log.Info.Printf(
					"Connected to observation mqtt broker: %s",
					env.SensorThingsObservationMqttUrl,
				)
			})
			opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
				log.Warning.Println("Connection to observation mqtt broker lost:", err)
			})
			randSource := rand.NewSource(time.Now().UnixNano())
			random := rand.New(randSource)
			clientID := fmt.Sprintf("priobike-udpmqttvisualizer-%d", random.Int())
			opts.SetClientID(clientID)
			opts.SetOrderMatters(false)
			opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
				log.Warning.Println("Received unexpected message on topic:", msg.Topic())
			})
			client = mqtt.NewClient(opts)
			if conn := client.Connect(); conn.Wait() && conn.Error() != nil {
				panic(conn.Error())
			}
		}

		wg.Add(1)
		go func(topic string) {
			defer wg.Done()

			// Subscribe to the datastream.
			if token := client.Subscribe(topic, observationQoS, func(client mqtt.Client, msg mqtt.Message) {
				// Process the message asynchronously to avoid blocking the mqtt client.
				go processMessage(msg)
			}); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		}(topic)
	}

	log.Info.Println("Subscribed to all datastreams.")
}
