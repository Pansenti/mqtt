package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
	"github.com/eclipse/paho.mqtt.golang"
)

var min_color int = 0
var min_intensity int = 0
var max_color int = 255
var max_intensity int = 240
var id string = "133755"
var have_controller bool = false
var color int = 200
var intensity int = 50
var mode string = "Manual"


var defaultMsgHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Unexpected TOPIC")
	fmt.Println("TOPIC:", msg.Topic())

	if len(msg.Payload()) > 0 {
		s := string(msg.Payload())
		fmt.Println("MSG  :", s)
	}
}

func updateStatus(client mqtt.Client) {
	val := fmt.Sprintf("%s:%d:%d", mode, color, intensity)

	if token := client.Publish("/fixture/status/" + id, 0, false, val); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}

func controllerGet(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("TOPIC:", msg.Topic())

	if len(msg.Payload()) > 0 {
		s := string(msg.Payload())
		fmt.Println("Payload Unexpectedd:", s)
	}

	updateStatus(client)
}

func controllerSet(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("TOPIC:", msg.Topic())

	if len(msg.Payload()) == 0 {
		return
	}

	s := string(msg.Payload())
	fmt.Println("MSG  :", s);

	if (s == "Auto") {
		mode = "Auto"
		// and do something
	} else {
		values := strings.Split(s, ":")

		if len(values) == 2 {
			i, err := strconv.Atoi(values[0])
			if err != nil {
				fmt.Println("Error parsing color:", err);
				return
			}

			j, err := strconv.Atoi(values[1])
			if err != nil {
				fmt.Println("Error parsing intensity:", err)
				return
			}

			color = i
			intensity = j
			mode = "Manual"
		}
	}
}

func controllerHello(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("TOPIC:", msg.Topic())

	if len(msg.Payload()) > 0 {
		s := string(msg.Payload())
		fmt.Println("MSG  :", s)
	}

	have_controller = true
	updateStatus(client)
}

func controllerGoodbye(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("TOPIC:", msg.Topic())

	if len(msg.Payload()) > 0 {
		s := string(msg.Payload())
		fmt.Println("MSG  :", s)
	}

	have_controller = false
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	if token := client.Subscribe("/controller/set/" + id, 0, controllerSet); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("/controller/set/all", 0, controllerSet); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("/controller/get/" + id, 0, controllerGet); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("/controller/hello/" + id, 0, controllerHello); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := client.Subscribe("/controller/goodbye", 0, controllerGoodbye); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func main() {
	// mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetDefaultPublishHandler(defaultMsgHandler)
	opts.SetOnConnectHandler(connectHandler)
	// this isn't working
	opts.SetWill("/fixture/goodbye/" + id, id, 0, false)

	client := mqtt.NewClient(opts)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	sig_received := make(chan os.Signal, 1)
	signal.Notify(sig_received, os.Interrupt)

	done := false

	for !done {
		select {
		case s := <-sig_received:
			fmt.Println("Received signal:", s)
			done = true
			if client.IsConnected() {
				// until SetWill works, force it
				if token := client.Publish("/fixture/goodbye/" + id, 0, false, id); token.Wait() && token.Error() != nil {
					fmt.Println(token.Error())
				}
			}
		case <-ticker.C:
			if client.IsConnected() {
				if have_controller {
					updateStatus(client)
				} else {
					if token := client.Publish("/fixture/hello/" + id, 0, false, id); token.Wait() && token.Error() != nil {
						fmt.Println(token.Error())
					}
				}
			} else {
				if token := client.Connect(); token.Wait() && token.Error() != nil {
					fmt.Println("Connect error", token.Error())
				}
			}
		}
	}

	client.Disconnect(500)
}

