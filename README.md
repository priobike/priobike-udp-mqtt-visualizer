# priobike-udp-mqtt-visualizer

A debugging tool that visualizes real-time states and predictions on a map.

Using the PrioBike deployment, the tool connects to the prediction-service's mqtt broker and to the observation mqtt broker (sending real-time traffic light data), and updates a json file every second that can then be displayed in the browser.

[Learn more about PrioBike](https://github.com/priobike)

## Quickstart

The easiest way to spin up a minimalistic setup is to use the provided docker-compose setup:

```
docker-compose up
```

Now, the map under `localhost/map.html` should update every second.

The `.env.hamburg` and `.env.dresden` files contain information needed to connect to the appropriate data brokers.

## Contributing

We highly encourage you to open an issue or a pull request. You can also use our repository freely with the `MIT` license. 

## Anything unclear?

Help us improve this documentation. If you have any problems or unclarities, feel free to open an issue.
