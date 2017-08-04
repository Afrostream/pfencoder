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
	initialized bool
	conn        *amqp.Connection     /* connection to the RabbitMQ server */
	ch          *amqp.Channel        /* channel in the RabbitMQ server */
	q           amqp.Queue           /* queue in the RabbitMQ server */
	msgs        <-chan amqp.Delivery /* messages to process from RabbitMQ server */
}

func newExchangerTask(rabbitmqHost string, rabbitmqPort int, rabbitmqUser string, rabbitmqPassword string) ExchangerTask {
	return (ExchangerTask{rabbitmqHost: rabbitmqHost,
		rabbitmqPort:     rabbitmqPort,
		rabbitmqUser:     rabbitmqUser,
		rabbitmqPassword: rabbitmqPassword})
}

func (e *ExchangerTask) init() {
	log.Printf("-- ExchangerTask init...")
	var err error
	first := true
	for first == true || err != nil {
		uri := fmt.Sprintf(`amqp://%s:%s@%s:%d`, e.rabbitmqUser, e.rabbitmqPassword, e.rabbitmqHost, e.rabbitmqPort)
		log.Println("connecting to RabbitMQ using uri=", uri)
		e.conn, err = amqp.Dial(uri)
		logOnError(err, "Waiting for RabbitMQ to become ready...")
		time.Sleep(1 * time.Second)
		first = false
	}
	//defer conn.Close()

	e.ch, err = e.conn.Channel()
	failOnError(err, "Failed to open a channel")
	//defer ch.Close()

	err = e.ch.ExchangeDeclare(
		"afsm-encoders", // name
		"fanout",        // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	e.q, err = e.ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	err = e.ch.QueueBind(
		e.q.Name,        // queue name
		"",              // routing key
		"afsm-encoders", // exchange
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")
	e.initialized = true
	log.Printf("-- ExchangerTask init done successfully")
}

func (e *ExchangerTask) start() {
	if e.initialized == false {
		log.Printf("ExchangerTask not initialized, Thread cannot start...")
		return
	}
	log.Printf("-- ExchangerTask Thread starting...")
	var err error
	e.msgs, err = e.ch.Consume(
		e.q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")
	/* Here we wait messages from the RabbitMQ to be processed */
	done := make(chan error)
	go func() {
		log.Printf("-- ExchangerTask Thread started")
		for d := range e.msgs {
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
		}
		log.Printf("-- ExchangerTask Thread stopped")
	}()
	done <- nil
}
