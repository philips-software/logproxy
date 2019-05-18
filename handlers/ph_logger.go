package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/influxdata/go-syslog/v2"
	"github.com/influxdata/go-syslog/v2/rfc5424"
	"github.com/m4rw3r/uuid"
	"github.com/philips-software/go-hsdp-api/logging"
	"github.com/streadway/amqp"
)

var (
	batchSize      = 25
	logTimeFormat  = "2006-01-02T15:04:05.000Z07:00"
	rtrTimeFormat  = "2006-01-02T15:04:05.000Z0700"
	ignorePatterns = []*regexp.Regexp{
		regexp.MustCompile(`Consul Health Check`),
		regexp.MustCompile(`POST /syslog/drain`),
		regexp.MustCompile(`GET /api/version`),
	}
	requestUsersAPIPattern = regexp.MustCompile(`/api/users/(?P<userID>[^\?/\s]+)`)
	vcapPattern            = regexp.MustCompile(`vcap_request_id:"(?P<requestID>[^"]+)"`)
	rtrPattern             = regexp.MustCompile(`\[RTR/(?P<index>\d+)\]`)
	rtrFormat              = regexp.MustCompile(`(?P<hostname>[^\?/\s]+) - \[(?P<time>[^\/\s]+)\]`)
)

type DHPLogMessage struct {
	Category            string     `json:"cat"`
	EventID             string     `json:"evt"`
	ApplicationVersion  string     `json:"ver"`
	Component           string     `json:"cmp"`
	ApplicationName     string     `json:"app"`
	ApplicationInstance string     `json:"inst"`
	ServerName          string     `json:"srv"`
	TransactionID       string     `json:"trns"`
	ServiceName         string     `json:"service"`
	LogTime             string     `json:"time"`
	OriginatingUser     string     `json:"usr"`
	Severity            string     `json:"sev"`
	LogData             DHPLogData `json:"val"`
}

type DHPLogData struct {
	Message string `json:"message"`
}

type Logger interface {
	Debugf(format string, args ...interface{})
}

type PHLogger struct {
	SharedKey    string
	SharedSecret string
	BaseURL      string
	ProductKey   string
	debug        bool
	client       *logging.Client
	parser       syslog.Machine
	log          Logger
}

func NewPHLogger(log Logger) (*PHLogger, error) {
	var logger PHLogger

	logger.SharedKey = os.Getenv("HSDP_LOGINGESTOR_KEY")
	logger.SharedSecret = os.Getenv("HSDP_LOGINGESTOR_SECRET")
	logger.BaseURL = os.Getenv("HSDP_LOGINGESTOR_URL")
	logger.ProductKey = os.Getenv("HSDP_LOGINGESTOR_PRODUCT_KEY")

	client, err := logging.NewClient(http.DefaultClient, logging.Config{
		SharedKey:    logger.SharedKey,
		SharedSecret: logger.SharedSecret,
		BaseURL:      logger.BaseURL,
		ProductKey:   logger.ProductKey,
	})
	if err != nil {
		return nil, err
	}
	logger.client = client
	logger.parser = rfc5424.NewParser()
	logger.log = log // Meta

	return &logger, nil
}

func (l *PHLogger) QueueName() string {
	return "logproxy4"
}

func (l *PHLogger) RFC5424QueueName() string {
	return "logproxy_rfc5424"
}

func (h *PHLogger) ackDelivery(d amqp.Delivery) {
	err := d.Ack(true)
	if err != nil {
		fmt.Printf("Error Acking delivery: %v\n", err)
	}
}

func (h *PHLogger) RFC5424Worker(deliveries <-chan amqp.Delivery) error {
	var count int
	var dropped int
	buf := make([]logging.Resource, batchSize)

	for {
		select {
		case d := <-deliveries:
			syslogMessage, err := h.parser.Parse(d.Body)
			if err != nil {
				fmt.Printf("Error parsing syslogMessage: %v\nBody: %s", err, string(d.Body))
				h.ackDelivery(d)
				continue
			}
			if syslogMessage == nil || syslogMessage.Message() == nil {
				fmt.Printf("No message in syslogMessage\n")
				h.ackDelivery(d)
				continue
			}
			resource, err := h.processMessage(syslogMessage)
			if err != nil {
				fmt.Printf("Error processing syslogMessage: %v\n", err)
				h.ackDelivery(d)
				continue
			}
			h.ackDelivery(d)
			if resource == nil { // Dropped message
				dropped++
				continue
			}
			buf[count] = *resource
			count++
			if count == batchSize {
				fmt.Printf("Batch sending %d messages\n", count)
				_, err := h.client.StoreResources(buf, count)
				if err != nil {
					fmt.Printf("Batch sending failed: %v\n", err)
				}
				count = 0
			}
		case <-time.After(1 * time.Second):
			if count > 0 {
				fmt.Printf("Batch sending %d messages (flush)\n", count)
				_, err := h.client.StoreResources(buf, count)
				if err != nil {
					fmt.Printf("Batch flushing failed: %v\n", err)
				}
				count = 0
			}
			if dropped > 0 {
				fmt.Printf("Dropped %d messages\n", dropped)
				dropped = 0
			}
		}
	}
}

func (h *PHLogger) processMessage(rfcLogMessage syslog.Message) (*logging.Resource, error) {
	var dhp DHPLogMessage
	var msg logging.Resource

	logMessage := rfcLogMessage.Message()
	if logMessage == nil {
		return nil, fmt.Errorf("no message found in syslog entry")
	}
	for _, i := range ignorePatterns {
		if i.MatchString(*logMessage) {
			if h.debug {
				h.log.Debugf("DROP --> %s\n", *logMessage)
			}
			return nil, nil
		}
	}

	if req := requestUsersAPIPattern.FindStringSubmatch(*logMessage); req != nil {
		if h.debug {
			h.log.Debugf("USR --> [%s] %s\n", req[1], *logMessage)
		}
		msg = h.wrapResource(req[1], rfcLogMessage)
		return &msg, nil
	}
	msg = h.wrapResource("logproxy", rfcLogMessage)
	err := json.Unmarshal([]byte(*logMessage), &dhp)
	if err == nil {
		if dhp.OriginatingUser != "" {
			msg.OriginatingUser = dhp.OriginatingUser
		}
		if dhp.TransactionID != "" {
			msg.TransactionID = dhp.TransactionID
		}
		if dhp.LogData.Message != "" {
			msg.LogData.Message = dhp.LogData.Message
		}
		if dhp.ApplicationVersion != "" {
			msg.ApplicationVersion = dhp.ApplicationVersion
		}
		if dhp.Component != "" {
			msg.Component = dhp.Component
		}
		if h.debug {
			h.log.Debugf("DHP --> %s\n", *logMessage)
		}
	}
	return &msg, nil

}

func (h *PHLogger) wrapResource(originatingUser string, msg syslog.Message) logging.Resource {
	var lm logging.Resource

	// ID
	id, _ := uuid.V4()
	lm.ID = id.String()

	// EventID
	lm.EventID = "1"

	// Category
	lm.Category = "ApplicationLog"

	// Component
	lm.Component = "logproxy"

	// TransactionID
	if vcap := vcapPattern.FindStringSubmatch(lm.LogData.Message); len(vcap) > 0 {
		lm.TransactionID = vcap[1]
	} else {
		uuid, _ := uuid.V4()
		lm.TransactionID = uuid.String()
	}

	// ServiceName
	lm.ServiceName = "logproxy"
	if a := msg.Appname(); a != nil {
		lm.ServiceName = *a
	}

	// ApplicationName
	lm.ApplicationName = "logproxy"
	if a := msg.Appname(); a != nil {
		lm.ApplicationName = *a
	}

	// ApplicationInstance
	lm.ApplicationInstance = "logproxy"
	if h := msg.Hostname(); h != nil {
		lm.ApplicationInstance = *h
	}

	// OrgiginatingUser
	lm.OriginatingUser = originatingUser

	// ApplicationVersion
	lm.ApplicationVersion = "1.0.0"

	// ServerName
	lm.ServerName = "logproxy"
	if s := msg.Hostname(); s != nil {
		lm.ServerName = *s
	}

	// Severity
	lm.Severity = "INFO"
	if sev := msg.SeverityLevel(); sev != nil {
		lm.Severity = *sev
	}

	// LogTime
	lm.LogTime = time.Now().Format(logTimeFormat)
	if m := msg.Timestamp(); m != nil {
		lm.LogTime = m.Format(logTimeFormat)
	}
	if procID := msg.ProcID(); procID != nil {
		if rtrPattern.FindStringSubmatch(*procID) != nil && msg.Message() != nil {
			m := msg.Message()
			if rtr := rtrFormat.FindStringSubmatch(*m); rtr != nil {
				if rtrTime, err := time.Parse(rtrTimeFormat, rtr[2]); err == nil {
					lm.LogTime = rtrTime.Format(logTimeFormat)
				}
			}
		}
	}

	// LogData
	lm.LogData.Message = "no message identified"
	if m := msg.Message(); m != nil {
		lm.LogData.Message = *m
		if p := msg.ProcID(); p != nil {
			lm.LogData.Message = fmt.Sprintf("%s %s", *p, *m)
		}
	}

	return lm
}
