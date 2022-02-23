import logging
from os import environ

import paho.mqtt.client as mqtt

bridge_host = environ.get("BRIDGE_HOST")
bridge_port = int(environ.get("BRIDGE_PORT"))

def on_bridge_connect(remote_client, userdata, flags, rc):
    logging.info(f"Connected to bridge with result code {rc}")

bridge_client = mqtt.Client(clean_session=True)
bridge_client.on_connect = on_bridge_connect
while True:
    try:
        bridge_client.connect(bridge_host, bridge_port, 60)
        break
    except Exception as e:
        pass

remote_host = environ.get("REMOTE_BROKER_HOST")
remote_port = int(environ.get("REMOTE_BROKER_PORT"))

def on_remote_connect(remote_client, userdata, flags, rc):
    logging.info(f"Connected to remote with result code {rc}")
    remote_client.subscribe("#", qos=0)

def on_remote_message(remote_client, userdata, msg):
    # Republish the message on the bridge client
    logging.info(f"Received message from remote: {msg.topic} {msg.payload}")
    bridge_client.publish(msg.topic, msg.payload, qos=0)

remote_client = mqtt.Client(clean_session=True)
remote_client.on_connect = on_remote_connect
remote_client.on_message = on_remote_message

remote_client.connect(remote_host, remote_port, 60)
remote_client.loop_forever()
