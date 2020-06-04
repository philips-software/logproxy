package main

import (
	"os"
	"os/signal"
	"path/filepath"

	"github.com/philips-software/logproxy/queue"
	"github.com/philips-software/logproxy/shared"

	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/philips-software/logproxy/handlers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/labstack/echo"

	"net/http"
	_ "net/http/pprof"
)

var commit = "deadbeaf"
var release = "v1.2.0"
var buildVersion = release + "-" + commit

func main() {
	e := make(chan *echo.Echo, 1)
	os.Exit(realMain(e))
}

func realMain(echoChan chan<- *echo.Echo) int {
	logger := log.New()

	viper.SetEnvPrefix("logproxy")
	viper.SetDefault("syslog", true)
	viper.SetDefault("ironio", false)
	viper.SetDefault("queue", "rabbitmq")
	viper.SetDefault("plugindir", "")
	viper.AutomaticEnv()

	enableIronIO := viper.GetBool("ironio")
	enableSyslog := viper.GetBool("syslog")
	queueType := viper.GetString("queue")
	token := os.Getenv("TOKEN")

	logger.Infof("logproxy %s booting", buildVersion)
	if !enableIronIO && !enableSyslog {
		logger.Errorf("both syslog and ironio drains are disabled")
		return 1
	}

	var messageQueue queue.Queue
	var err error

	// Queue Type
	switch queueType {
	case "rabbitmq":
		messageQueue, err = queue.NewRabbitMQQueue()
		if err != nil {
			logger.Errorf("RabbitMQ queue error: %v", err)
			return 128
		}
	default:
		messageQueue, _ = queue.NewChannelQueue()
		logger.Info("using internal channel queue")
	}

	// Echo framework
	e := echo.New()
	healthHandler := handlers.HealthHandler{}
	e.GET("/health", healthHandler.Handler())
	e.GET("/api/version", handlers.VersionHandler(buildVersion))

	// Syslog
	if enableSyslog {
		syslogHandler, err := handlers.NewSyslogHandler(token, messageQueue)
		if err != nil {
			logger.Errorf("failed to setup SyslogHandler: %s", err)
			return 3
		}
		logger.Info("enabling /syslog/drain/:token")
		e.POST("/syslog/drain/:token", syslogHandler.Handler())
	}

	// IronIO
	if enableIronIO {
		ironIOHandler, err := handlers.NewIronIOHandler(token, messageQueue)
		if err != nil {
			logger.Errorf("Failed to setup IronIOHandler: %s", err)
			return 4
		}
		logger.Info("enabling /ironio/drain/:token")
		e.POST("/ironio/drain/:token", ironIOHandler.Handler())
	}

	setupPprof(logger)
	setupInterrupts(logger)

	// Consumer
	var done chan bool
	if done, err = messageQueue.Start(); err != nil {
		logger.Errorf("failed to start consumer: %v", err)
		return 5
	}

	// Plugin Manager
	homeDir, _ := os.UserHomeDir()
	pluginExePath, _ := os.Executable()
	pluginManager := &shared.PluginManager{
		PluginDirs: []string{
			filepath.Join(homeDir, ".logproxy/plugins"),
			pluginExePath,
		},
	}
	if pluginDir := viper.GetString("plugindir"); pluginDir != "" {
		pluginManager.PluginDirs = append(pluginManager.PluginDirs, pluginDir)
	}
	if err := pluginManager.Discover(); err == nil {
		_ = pluginManager.LoadAll()
	}

	// Worker
	deliverer, err := setupDeliverer(http.DefaultClient, logger, pluginManager, buildVersion)
	if err != nil {
		logger.Errorf("failed to setup Deliverer: %s", err)
		return 20
	}
	doneWorker := make(chan bool)
	go deliverer.ResourceWorker(messageQueue, doneWorker)

	echoChan <- e
	exitCode := 0
	if err := e.Start(listenString()); err != nil {
		logger.Errorf(err.Error())
		exitCode = 6
	}
	done <- true
	doneWorker <- true
	return exitCode
}

func setupDeliverer(httpClient *http.Client, logger *log.Logger, manager *shared.PluginManager, buildVersion string) (*queue.Deliverer, error) {
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
	return queue.NewDeliverer(storer, logger, manager, buildVersion)
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
			os.Exit(0)
		}
	}()
}

func setupPprof(logger *log.Logger) {
	go func() {
		logger.Info("start pprof on localhost:6060")
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
