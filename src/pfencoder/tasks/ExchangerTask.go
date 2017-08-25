package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"os"
	"pfencoder/tools"
	"strconv"
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
	/* For test purpose only **/
	transcodingVersion int
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
	e.amqpUri = fmt.Sprintf("amqp://%s:%s@%s:%d", e.rabbitmqUser, e.rabbitmqPassword, e.rabbitmqHost, e.rabbitmqPort)
	if os.Getenv("TRANSCODING_VERSION") != "" {
		e.transcodingVersion, _ = strconv.Atoi(os.Getenv("TRANSCODING_VERSION"))
	}
	log.Printf("-- ExchangerTask init : TRANSCODING_VERSION=%d", e.transcodingVersion)
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
	go func() {
		for { //reconnection loop
			log.Printf("-- ExchangerTask : (Re)Connection to RabbitMQ...")
			done := false
			/* setup */
			conn := e.connectToRabbitMQ()
			defer conn.Close()
			notify := conn.NotifyClose(make(chan *amqp.Error))
			log.Printf("-- ExchangerTask : opening channel...")
			ch, err := conn.Channel()
			tools.LogOnError(err, "ExchangerTask : Failed to open a channel")
			if err == nil {
				defer ch.Close()
				log.Printf("-- ExchangerTask : opening channel done successfully")
			}
			var msgs <-chan amqp.Delivery
			if err == nil {
				log.Printf("-- ExchangerTask : declaring an exchange...")
				err = ch.ExchangeDeclare(
					"afsm-encoders", // name
					"fanout",        // type
					true,            // durable
					false,           // auto-deleted
					false,           // internal
					false,           // no-wait
					nil,             // arguments
				)
				tools.LogOnError(err, "ExchangerTask : Failed to declare an exchange")
				if err == nil {
					log.Printf("-- ExchangerTask : declaring an exchange done successfully")
				}
			}
			var q amqp.Queue
			if err == nil {
				log.Printf("-- ExchangerTask : declaring a queue...")
				q, err = ch.QueueDeclare(
					"",
					false,
					false,
					true,
					false,
					nil,
				)
				tools.LogOnError(err, "ExchangerTask : Failed to declare a queue")
				if err == nil {
					log.Printf("-- ExchangerTask : declaring an queue done successfully")
				}
			}
			if err == nil {
				log.Printf("-- ExchangerTask : binding a queue...")
				err = ch.QueueBind(
					q.Name,          // queue name
					"",              // routing key
					"afsm-encoders", // exchange
					false,
					nil,
				)
				tools.LogOnError(err, "ExchangerTask : Failed to bind a queue")
				if err == nil {
					log.Printf("-- ExchangerTask : binding a queue done successfully")
				}
			}
			if err == nil {
				log.Printf("-- ExchangerTask : registring a consumer...")
				msgs, err = ch.Consume(
					q.Name,
					"",
					true,
					false,
					false,
					false,
					nil,
				)
				tools.LogOnError(err, "ExchangerTask : Failed to register a consumer")
				if err == nil {
					log.Printf("-- ExchangerTask : registring a consumer done successfully")
				}
			}
			if err == nil {
				done = true
			}
			if done {
				log.Printf("-- ExchangerTask : (Re)Connection to RabbitMQ done successfully")
			} else {
				log.Printf("ExchangerTask : (Re)Connection to RabbitMQ failed")
			}
			/* wait for messages */
		MSGSLOOP:
			for {
				select {
				case failedError := <-notify:
					//work with error
					tools.LogOnError(failedError, "ExchangerTask : Lost connection to the RabbitMQ, will retry connection...")
					break MSGSLOOP //reconnect
				case msg := <-msgs:
					//work with message
					log.Printf("-- ExchangerTask : Receiving a message...")
					log.Printf("-- ExchangerTask : Received a message: %s", msg.Body)
					var oMessage OrderMessage
					err = json.Unmarshal([]byte(msg.Body), &oMessage)
					if oMessage.Hostname == e.hostname {
						//TODO : NCO : Why this limit, what happens if task is not taken ?
						if ffmpegProcesses < 4 {
							log.Printf("-- ExchangerTask : TranscoderTask creating...")
							transcoderTask := NewTranscoderTask(oMessage.AssetId)
							transcoderTask.Init()
							log.Printf("-- ExchangerTask : TranscoderTask inited, starting Encoding, transcodingVersion=%d", e.transcodingVersion)
							if e.transcodingVersion == 1 {
								go transcoderTask.StartEncoding()
							} else {
								go transcoderTask.DoEncoding()
							}
							log.Printf("-- ExchangerTask : TranscoderTask creating done successfully")
						} else {
							log.Printf("ExchangerTask : Cannot start one more ffmpeg process (encoding queue full)")
						}
					} else {
						log.Printf("-- ExchangerTask : message ignored : hostame=%s, message.hostname=%s", e.hostname, oMessage.Hostname)
					}
					log.Printf("-- ExchangerTask : Receiving a message done successfully")
				}
			}
		}
		log.Printf("ExchangerTask Thread stopped")
	}()
}

func (e *ExchangerTask) connectToRabbitMQ() *amqp.Connection {
	for {
		conn, err := amqp.Dial(e.amqpUri)
		if err == nil {
			return conn
		}
		tools.LogOnError(err, "ExchangerTask : Failed to connect to the RabbitMQ, retrying...")
		time.Sleep(3 * time.Second)
	}
}
