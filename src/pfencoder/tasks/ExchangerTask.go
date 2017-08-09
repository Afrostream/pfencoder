package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"os"
	"pfencoder/tools"
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
	hostname    string
}

func NewExchangerTask(rabbitmqHost string, rabbitmqPort int, rabbitmqUser string, rabbitmqPassword string) ExchangerTask {
	return (ExchangerTask{rabbitmqHost: rabbitmqHost,
		rabbitmqPort:     rabbitmqPort,
		rabbitmqUser:     rabbitmqUser,
		rabbitmqPassword: rabbitmqPassword})
}

func (e *ExchangerTask) Init() bool {
	log.Printf("-- ExchangerTask init starting...")
	var err error
	e.hostname, err = os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	e.amqpUri = fmt.Sprintf(`amqp://%s:%s@%s:%d`, e.rabbitmqUser, e.rabbitmqPassword, e.rabbitmqHost, e.rabbitmqPort)
	e.initialized = true
	log.Printf("-- ExchangerTask init done successfully")
	return e.initialized
}

func (e *ExchangerTask) Start() {
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
		tools.LogOnError(err, "Failed to open a channel")
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
			tools.LogOnError(err, "Failed to declare an exchange")
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
			tools.LogOnError(err, "Failed to declare a queue")
		}
		if err != nil {
			err = ch.QueueBind(
				q.Name,          // queue name
				"",              // routing key
				"afsm-encoders", // exchange
				false,
				nil,
			)
			tools.LogOnError(err, "Failed to bind a queue")
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
			tools.LogOnError(err, "Failed to register a consumer")
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
				tools.LogOnError(err, "Lost connection to the RabbitMQ, will retry connection...")
				break MSGSLOOP //reconnect
			case d := <-msgs:
				//work with message
				log.Printf("-- Receiving a message...")
				log.Printf("-- Received a message: %s", d.Body)
				var oMessage OrderMessage
				err = json.Unmarshal([]byte(d.Body), &oMessage)
				if oMessage.Hostname == e.hostname {
					if ffmpegProcesses < 4 {
						log.Printf("-- Start running ffmpeg process")
						transcoderTask := TranscoderTask{}
						go transcoderTask.DoEncoding(oMessage.AssetId)
						log.Printf("-- Func doEncoding() thread created")
					} else {
						log.Printf("Cannot start one more ffmpeg process (encoding queue full)")
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
		tools.LogOnError(err, "Failed to connect to the RabbitMQ, retrying...")
		time.Sleep(3 * time.Second)
	}

}
