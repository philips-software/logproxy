package main

import (
	"os"
	"os/signal"

	"github.com/loafoe/go-rabbitmq"
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/philips-software/logproxy/handlers"
	log "github.com/sirupsen/logrus"

	"github.com/labstack/echo"

	"net/http"
	_ "net/http/pprof"
)

var commit = "deadbeaf"
var release = "v1.1.0"
var buildVersion = release + "-" + commit

func main() {
	logger := log.New()
	logger.Infof("logproxy %s booting", buildVersion)

	// PHLogger
	phLogger, err := setupPHLogger(http.DefaultClient, logger)
	if err != nil {
		logger.Errorf("Failed to setup PHLogger: %s", err)
		os.Exit(1)
	}

	// RabbitMQ
	consumer, producer, err := setupConsumerProducer(phLogger)
	if err != nil {
		logger.Errorf("Failed to create consumer/producer: %v", err)
		os.Exit(2)
	}

	// Syslog
	syslogHandler, err := handlers.NewSyslogHandler(os.Getenv("TOKEN"), producer)
	if err != nil {
		logger.Errorf("Failed to setup SyslogHandler: %s", err)
		os.Exit(1)
	}

	// IronIO
	ironIOHandler, err := handlers.NewIronIOHandler(os.Getenv("TOKEN"), producer)
	if err != nil {
		logger.Errorf("Failed to setup IronIOHandler: %s", err)
		os.Exit(1)
	}

	// Echo framework
	e := echo.New()
	healthHandler := handlers.HealthHandler{}
	e.GET("/health", healthHandler.Handler())
	e.GET("/api/version", handlers.VersionHandler(buildVersion))
	e.POST("/syslog/drain/:token", syslogHandler.Handler())
	e.POST("/ironio/drain/:token", ironIOHandler.Handler())

	setupPprof(logger)
	setupInterrupts(logger)

	if err := consumer.Start(); err != nil {
		logger.Errorf("Failed to start consumer: %v", err)
		os.Exit(2)
	}
	if err := e.Start(listenString()); err != nil {
		logger.Errorf(err.Error())
	}
}

func setupConsumerProducer(phLogger *handlers.PHLogger) (*rabbitmq.AMQPConsumer, rabbitmq.Producer, error) {
	consumer, err := rabbitmq.NewConsumer(rabbitmq.Config{
		RoutingKey:   handlers.RoutingKey,
		Exchange:     handlers.Exchange,
		ExchangeType: "topic",
		Durable:      false,
		AutoDelete:   true,
		QueueName:    phLogger.RFC5424QueueName(),
		CTag:         consumerTag(),
		HandlerFunc:  phLogger.RFC5424Worker,
	})
	if err != nil {
		return nil, nil, err
	}
	producer, err := rabbitmq.NewProducer(rabbitmq.Config{
		Exchange:     handlers.Exchange,
		ExchangeType: "topic",
		Durable:      false,
	})
	if err != nil {
		return nil, nil, err
	}
	return consumer, producer, nil
}

func setupPHLogger(httpClient *http.Client, logger *log.Logger) (*handlers.PHLogger, error) {
	sharedKey := os.Getenv("HSDP_LOGINGESTOR_KEY")
	sharedSecret := os.Getenv("HSDP_LOGINGESTOR_SECRET")
	baseURL := os.Getenv("HSDP_LOGINGESTOR_URL")
	productKey := os.Getenv("HSDP_LOGINGESTOR_PRODUCT_KEY")

	storer, err := logging.NewClient(httpClient, logging.Config{
		SharedKey:    sharedKey,
		SharedSecret: sharedSecret,
		BaseURL:      baseURL,
		ProductKey:   productKey,
	})
	if err != nil {
		return nil, err
	}
	return handlers.NewPHLogger(storer, logger)
}

func setupInterrupts(logger *log.Logger) {
	// Setup a channel to receive a signal
	done := make(chan os.Signal, 1)

	// Notify this channel when a SIGINT is received
	signal.Notify(done, os.Interrupt)

	// Fire off a goroutine to loop until that channel receives a signal.
	// When a signal is received simply exit the program
	go func() {
		for range done {
			logger.Error("Exiting because of CTRL-C")
			os.Exit(0)
		}
	}()
}

func setupPprof(logger *log.Logger) {
	go func() {
		logger.Info("Start pprof on localhost:6060")
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			logger.Errorf("pprof not started: %v", err)
		}
	}()
}

func listenString() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return (":" + port)
}

func consumerTag() string {
	return "logproxy"
}
