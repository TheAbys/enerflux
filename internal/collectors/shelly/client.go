package shelly

import (
	"encoding/json"
	"fmt"
	"strings"

	paho "github.com/eclipse/paho.mqtt.golang"
)

const switchStatusTopic = "+/status/switch:0"

type SwitchStatusHandler func(deviceID string, status SwitchStatus)

type Client interface {
	Connect() error
	SubscribeSwitchStatus(handler SwitchStatusHandler) error
	Close()
}

type MQTTClient struct {
	client              paho.Client
	switchStatusHandler SwitchStatusHandler
}

func NewClient(brokerURL, clientID string) *MQTTClient {
	c := &MQTTClient{}

	opts := paho.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID).
		SetAutoReconnect(true)

	opts.SetOnConnectHandler(func(_ paho.Client) {
		if c.switchStatusHandler != nil {
			_ = c.subscribeSwitchStatus()
		}
	})

	c.client = paho.NewClient(opts)

	return c
}

func (c *MQTTClient) Connect() error {
	token := c.client.Connect()
	token.Wait()

	if err := token.Error(); err != nil {
		return fmt.Errorf("connect mqtt: %w", err)
	}

	return nil
}

func (c *MQTTClient) SubscribeSwitchStatus(handler SwitchStatusHandler) error {
	c.switchStatusHandler = handler
	return c.subscribeSwitchStatus()
}

func (c *MQTTClient) subscribeSwitchStatus() error {
	token := c.client.Subscribe(
		switchStatusTopic,
		1,
		func(_ paho.Client, msg paho.Message) {
			var status SwitchStatus

			if err := json.Unmarshal(msg.Payload(), &status); err != nil {
				return
			}

			deviceID := strings.TrimSuffix(
				msg.Topic(),
				"/status/switch:0",
			)

			c.switchStatusHandler(deviceID, status)
		},
	)

	token.Wait()

	if err := token.Error(); err != nil {
		return fmt.Errorf("subscribe %q: %w", switchStatusTopic, err)
	}

	return nil
}

func (c *MQTTClient) Close() {
	c.client.Disconnect(250)
}
