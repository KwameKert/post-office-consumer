package app

import (
	"encoding/json"
	"flag"
	"office-consumer/app/core"
	"office-consumer/app/database"
	"os"
	"os/signal"
	"syscall"
	"time"

	logModule "office-consumer/app/log"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"

	amqp "github.com/rabbitmq/amqp091-go"
)

type App struct{}

var (
	uri               = flag.String("uri", "amqp://guest:guest@localhost:5672/", "AMQP URI")
	exchange          = flag.String("exchange", "test-exchange", "Durable, non-auto-deleted AMQP exchange name")
	exchangeType      = flag.String("exchange-type", "direct", "Exchange type - direct|fanout|topic|x-custom")
	queue             = flag.String("queue", "test-queue", "Ephemeral AMQP queue name")
	bindingKey        = flag.String("key", "test-key", "AMQP binding key")
	consumerTag       = flag.String("consumer-tag", "simple-consumer", "AMQP consumer tag (should not be blank)")
	lifetime          = flag.Duration("lifetime", 500*time.Second, "lifetime of process before shutdown (0s=infinite)")
	verbose           = flag.Bool("verbose", true, "enable verbose output of message data")
	autoAck           = flag.Bool("auto_ack", false, "enable message auto-ack")
	deliveryCount int = 0
)
var connection *mongo.Client
var logService *logModule.LogServiceLayer

func (app *App) Start(conf *core.Config) {

	log.Info("Connecting database")
	connection = setupDatabase(conf)
	repo := logModule.NewLogRepoLayer(connection.Database("agerp-post-office"))
	logService = logModule.NewLogServiceLayer(*repo)
	log.Info("starting message broker")
	c, err := NewConsumer(*uri, *exchange, *exchangeType, *queue, *bindingKey, *consumerTag)
	if err != nil {
		log.Fatal(err)
	}

	SetupCloseHandler(c)

	if *lifetime > 0 {
		log.Info("running for ", *lifetime)
		time.Sleep(*lifetime)
	} else {
		log.Info("running until Consumer is done")
		<-c.done
	}

	log.Info("shutting down")

	if err := c.Shutdown(); err != nil {
		log.Fatalf("error during shutdown: ", err)
	}
}

func setupDatabase(conf *core.Config) *mongo.Client {
	mg, err := database.GetMongoClient(conf)
	if err != nil {
		log.Fatal("failed to initialize postgres database. err:", err)
		panic(err.Error())
	}
	if err != nil {
		log.Fatal("failed to run migrations. err:", err)
	}

	return mg
}

func init() {
	flag.Parse()
}

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
}

func NewConsumer(amqpURI, exchange, exchangeType, queueName, key, ctag string) (*Consumer, error) {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
	}

	var err error

	config := amqp.Config{Properties: amqp.NewConnectionProperties()}
	config.Properties.SetClientConnectionName("sample-consumer")
	log.Info("dialing ", amqpURI)
	c.conn, err = amqp.DialConfig(amqpURI, config)
	if err != nil {
		log.Error("Dial: ", err)
		return nil, err
	}

	go func() {
		log.Info("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	log.Info("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		log.Error("Channel: ", err)
		return nil, err
	}

	// log.Info("got Channel, declaring Exchange (%q)", exchange)
	// if err = c.channel.ExchangeDeclare(
	// 	exchange,     // name of the exchange
	// 	exchangeType, // type
	// 	true,         // durable
	// 	false,        // delete when complete
	// 	false,        // internal
	// 	false,        // noWait
	// 	nil,          // arguments
	// ); err != nil {
	// 	return nil, err
	// }

	log.Info("declared Exchange, declaring Queue ", queueName)
	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		log.Error("Queue Declare: ", err)
		return nil, err
	}

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,        // arguments
	); err != nil {
		log.Error("Queue Bind: %s", err)
		return nil, err
	}

	log.Info("Queue bound to Exchange, starting Consume (consumer tag %q)", c.tag)
	deliveries, err := c.channel.Consume(
		"test-queue", // name
		"",           // consumerTag,
		*autoAck,     // autoAck
		false,        // exclusive
		false,        // noLocal
		false,        // noWait
		nil,          // arguments
	)
	if err != nil {
		log.Error("Queue Consume: ", err)
		return nil, err
	}

	go handle(deliveries, c.done)

	return c, nil
}

func (c *Consumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		log.Error("Consumer cancel failed: ", err)
		return err
	}

	if err := c.conn.Close(); err != nil {
		log.Error("AMQP connection close error: %s", err)
		return err
	}

	defer log.Error("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}

func handle(deliveries <-chan amqp.Delivery, done chan error) {
	cleanup := func() {
		log.Error("handle: deliveries channel closed")
		done <- nil
	}

	defer cleanup()

	for d := range deliveries {
		deliveryCount++
		if *verbose {
			//fmt.Printf("Recieved Message: %s\n", d.Body)
			var req logModule.Log
			json.Unmarshal(d.Body, &req)
			logService.AddLog(req)

		} else {
			if deliveryCount%65536 == 0 {
				log.Info("delivery count %d", deliveryCount)
			}
		}
		if !*autoAck {
			d.Ack(false)
		}
	}
}

func SetupCloseHandler(consumer *Consumer) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("Ctrl+C pressed in Terminal")
		if err := consumer.Shutdown(); err != nil {
			log.Fatal("error during shutdown: %s", err)
		}
		os.Exit(0)
	}()
}

type Data struct {
	Text        string `json:"text"`
	Destination string `json:"destination"`
	SenderId    string `json:"senderId"`
	Type        string `json:"type"`
}
