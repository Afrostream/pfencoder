package main

import (
	"fmt"
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
	"os"
	"time"
)

type ExchangerTask struct {
	rabbitmqHost string
	rabbitmqUser string
	rabbitmqPassword string
}

func (e ExchangerTask) startTask() {
	var conn *amqp.Connection
	 var err error
	first := true
	for first == true || err != nil {
		conn, err = amqp.Dial(fmt.Sprintf(`amqp://%s:%s@%s/`, e.rabbitmqUser, e.rabbitmqPassword, e.rabbitmqHost))
		logOnError(err, "Waiting for RabbitMQ to become ready...")
		time.Sleep(1 * time.Second)
		first = false
	}
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"afsm-encoders", // name
		"fanout",        // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.QueueBind(
		q.Name,          // queue name
		"",              // routing key
		"afsm-encoders", // exchange
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			var oMessage OrderMessage
			err = json.Unmarshal([]byte(d.Body), &oMessage)
			hostname, err := os.Hostname()
			if err != nil {
				log.Fatal(err)
			} else {
				if oMessage.Hostname == hostname {
					if ffmpegProcesses < 4 {
						log.Printf("Start running ffmpeg process")
						transcoderTask := TranscoderTask{}
						go transcoderTask.doEncoding(oMessage.AssetId)
						log.Printf("Func doEncoding() thread created")
					} else {
						log.Printf("Cannot start one more ffmpeg process (encoding queue full)")
					}
				}
			}
		}
	}()

	log.Printf(" [*] Waiting for messages, To exit press CTRL+C")
	<-forever
}
