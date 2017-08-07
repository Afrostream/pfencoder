package main

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"os"
	"time"
)

type ExchangerTask struct {
	/* constructor */
	rabbitmqHost     string
	rabbitmqPort     int
	rabbitmqUser     string
	rabbitmqPassword string
	/* 'instance' variables */
	amqpUri     string
	initialized bool
}

func newExchangerTask(rabbitmqHost string, rabbitmqPort int, rabbitmqUser string, rabbitmqPassword string) ExchangerTask {
	return (ExchangerTask{rabbitmqHost: rabbitmqHost,
		rabbitmqPort:     rabbitmqPort,
		rabbitmqUser:     rabbitmqUser,
		rabbitmqPassword: rabbitmqPassword})
}

func (e *ExchangerTask) init() {
	log.Printf("-- ExchangerTask init starting...")
	e.amqpUri = fmt.Sprintf(`amqp://%s:%s@%s:%d`, e.rabbitmqUser, e.rabbitmqPassword, e.rabbitmqHost, e.rabbitmqPort)
	e.initialized = true
	log.Printf("-- ExchangerTask init done successfully")
}

func (e *ExchangerTask) start() {
	if e.initialized == false {
		log.Printf("ExchangerTask not initialized, Thread cannot start...")
		return
	}
	log.Printf("-- ExchangerTask Thread starting...")
	for { //reconnection loop
		log.Printf("-- (Re)Connection to RabbitMQ...")
		done := false
		/* setup */
		conn := e.connectToRabbitMQ()
		defer conn.Close()
		notify := conn.NotifyClose(make(chan *amqp.Error))
		ch, err := conn.Channel()
		logOnError(err, "Failed to open a channel")
		var msgs <-chan amqp.Delivery
		if err != nil {
			defer ch.Close()
		}
		if err != nil {
			err = ch.ExchangeDeclare(
				"afsm-encoders", // name
				"fanout",        // type
				true,            // durable
				false,           // auto-deleted
				false,           // internal
				false,           // no-wait
				nil,             // arguments
			)
			logOnError(err, "Failed to declare an exchange")
		}
		var q amqp.Queue
		if err != nil {
			q, err = ch.QueueDeclare(
				"",
				false,
				false,
				true,
				false,
				nil,
			)
			logOnError(err, "Failed to declare a queue")
		}
		if err != nil {
			err = ch.QueueBind(
				q.Name,          // queue name
				"",              // routing key
				"afsm-encoders", // exchange
				false,
				nil,
			)
			logOnError(err, "Failed to bind a queue")
		}
		if err != nil {
			msgs, err = ch.Consume(
				q.Name,
				"",
				true,
				false,
				false,
				false,
				nil,
			)
			logOnError(err, "Failed to register a consumer")
		}
		if err == nil {
			done = true
		}
		if done {
			log.Printf("-- (Re)Connection to RabbitMQ done successfully")
		} else {
			log.Printf("(Re)Connection to RabbitMQ failed")
		}
		/* wait for messages */
	MSGSLOOP:
		for {
			select {
			case err := <-notify:
				//work with error
				logOnError(err, "Lost connection to the RabbitMQ, will retry connection...")
				break MSGSLOOP //reconnect
			case d := <-msgs:
				//work with message
				log.Printf("-- Receiving a message...")
				log.Printf("-- Received a message: %s", d.Body)
				var oMessage OrderMessage
				err := json.Unmarshal([]byte(d.Body), &oMessage)
				hostname, err := os.Hostname()
				if err != nil {
					log.Fatal(err)
				} else {
					if oMessage.Hostname == hostname {
						if ffmpegProcesses < 4 {
							log.Printf("-- Start running ffmpeg process")
							transcoderTask := TranscoderTask{}
							go transcoderTask.doEncoding(oMessage.AssetId)
							log.Printf("-- Func doEncoding() thread created")
						} else {
							log.Printf("Cannot start one more ffmpeg process (encoding queue full)")
						}
					}
				}
				log.Printf("-- Receiving a message done successfully")
			}
		}
	}
	log.Printf("ExchangerTask Thread stopped")
}

func (e *ExchangerTask) connectToRabbitMQ() *amqp.Connection {
	for {
		conn, err := amqp.Dial(e.amqpUri)
		if err == nil {
			return conn
		}
		logOnError(err, "Failed to connect to the RabbitMQ, retrying...")
		time.Sleep(3 * time.Second)
	}

}
