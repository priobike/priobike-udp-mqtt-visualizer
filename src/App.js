import { ScatterplotLayer } from '@deck.gl/layers';
import DeckGL from '@deck.gl/react';
import AppBar from '@mui/material/AppBar';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Toolbar from '@mui/material/Toolbar';
import Typography from '@mui/material/Typography';
import React from 'react';
import {BASEMAP} from '@deck.gl/carto';
import CircularProgress from '@mui/material/CircularProgress';
import { StaticMap as Map } from 'react-map-gl';
import './App.css';

const mqttBackend = process.env.REACT_APP_MQTT_URL;
const frostBackend = process.env.REACT_APP_FROST_URL;
const thingsQuery = process.env.REACT_APP_THINGS_QUERY;

console.log(`Observations will be retrieved from: ${mqttBackend}`);
console.log(`Things will be retrieved from: ${frostBackend}`);

const INITIAL_VIEW_STATE = {
  // Center of Hamburg
  longitude: 9.99,
  latitude: 53.55,
  zoom: 12,
  pitch: 0,
  bearing: 0
};

const mqtt = require('mqtt')

export default class App extends React.Component {
  constructor(props) {
    super(props);

    this.mqttClient = null;

    this.state = {
      loadingText: null,
      mqttMessages: 0,
      things: {},
      datastreams: {},
      observations: {},
    }
  }

  componentDidMount() {
    this.loadThings();
    this.connectMqttClient();
  }

  connectMqttClient = () => {
    console.log(`Connecting to MQTT backend: ${mqttBackend}`);
    this.mqttClient = mqtt.connect(mqttBackend, {
      clean: true,
      clientId: 'priobike-udp-mqtt-visualizer',
    });
    this.mqttClient.on('connect', function () {
      console.log(`Connected to MQTT backend: ${mqttBackend}`);
    });
    this.mqttClient.on('message', (topic, message) => {   
      const { datastreams } = this.state;
      if (datastreams) {
        const thingId = datastreams[topic];
        if (thingId) {
          const observation = JSON.parse(message.toString());
          const result = observation.result;
          this.setState(oldState => {
            const observations = { ...oldState.observations };
            observations[thingId] = result;
            return { observations, mqttMessages: oldState.mqttMessages + 1 };
          });
        }
      }
    });
    this.mqttClient.subscribe('#', {
      qos: 1,
    })
  }

  async loadThings() {
    let url = `${frostBackend}${thingsQuery}`;
    let newThings = {};
    let newDatastreams = {};
    do {
      this.setState({ loadingText: `Loading ${url}` });
      console.log("Loading things from: " + url);
      const response = await fetch(url);
      const json = await response.json();
      url = json['@iot.nextLink'];
      const listOfThings = json.value;
      for (let thing of listOfThings) {
        newThings[thing.name] = thing;
        for (let datastream of thing.Datastreams) {
          let name = datastream.name;
          if (name.startsWith('Primary signal heads at ')) {
            let id = datastream['@iot.id'];
            newDatastreams[`v1.1/Datastreams(${id})/Observations`] = thing.name;
          }
        }
      }
    } while (url);
    console.log("Loaded things: ", newThings);
    console.log("Loaded datastreams: ", newDatastreams);
    this.setState({ things: newThings, datastreams: newDatastreams, loadingText: null });
  }
  
  render() {
    const { things, observations } = this.state;

    let loadingIndicator = (null);
    if (this.state.loadingText) {
      loadingIndicator = (
        <Stack direction="row" alignItems="center" spacing={4}>
          <Typography variant="small" noWrap component="div">{this.state.loadingText}</Typography>
          <CircularProgress style={{'color': 'white'}} />
        </Stack>
      );
    }

    const thingIds = Object.keys(things);
    const layers = [
      new ScatterplotLayer({
        id: 'scatterplot-layer',
        data: thingIds,
        pickable: true,
        opacity: 0.8,
        stroked: true,
        filled: true,
        radiusScale: 6,
        radiusMinPixels: 7,
        radiusMaxPixels: 100,
        lineWidthMinPixels: 1,
        getPosition: thingId => things[thingId]['Locations'][0]['location']['geometry']['coordinates'][1][0],
        getRadius: d => 3,
        getFillColor: thingId => {
          const observation = observations[thingId];
          if (observation) {
            if (observation === 0) {
              // Dark
              return [0, 0, 0, 255];
            } else if (observation === 1) {
              // Red
              return [255, 0, 0, 255];
            } else if (observation === 2) {
              // Amber
              return [255, 255, 0, 255];
            } else if (observation === 3) {
              // Green
              return [0, 255, 0, 255];
            } else if (observation === 4) {
              // Red-Amber
              return [255, 128, 0, 255];
            } else if (observation === 5) {
              // Amber-Flashing
              return [255, 255, 0, 255];
            } else if (observation === 6) {
              // Green-Flashing
              return [0, 255, 0, 255];
            }
          }
          return [0, 0, 0, 0];
        },
        getLineColor: d => [0, 0, 0, 255],
      })
    ];

    return (
      <Box
        component="main"
        sx={{ flexGrow: 1, p: 0 }}
      >
        <AppBar>
        <Toolbar>
          <Stack direction="row" spacing={2} alignItems="center">
            <Typography variant="h6" noWrap component="div">
              UDP MQTT Visualizer
            </Typography>
            <Typography variant="p" noWrap component="div">
              Messages: {this.state.mqttMessages}
            </Typography>
            {loadingIndicator}
          </Stack>
        </Toolbar>
        </AppBar>
        <DeckGL
          style={{ 
            height: '95vh', 
            position: 'relative' 
          }}
          initialViewState={INITIAL_VIEW_STATE}
          controller={true}
          layers={layers}
        >
          <Map mapStyle={BASEMAP.POSITRON} />
        </DeckGL>
      </Box>
    );
  }
}
