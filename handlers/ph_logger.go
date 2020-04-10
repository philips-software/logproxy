package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/influxdata/go-syslog/v2"
	"github.com/influxdata/go-syslog/v2/rfc5424"
	"github.com/m4rw3r/uuid"
	"github.com/philips-software/go-hsdp-api/logging"
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

	errNoMessage = errors.New("No message in syslogMessage")

	defaultInvalidCharacters             = "$&+,:;=?@#|<>()[]"
	applicationNameInvalidCharacters     = "$&+,;=?@#|<>()[]"
	applicationInstanceInvalidCharacters = "[&+;?@|<>()[]"
	eventIDInvalidCharacters             = "&+,:;=?@#|<>()[]"
	otherNameInvalidCharacters           = "&+,;=?@#|<>()[]"
	originatingUsersInvalidCharacters    = "$&+;=?@#|<>()[]"
	versionInvalidCharacters             = "&+;=?@|<>()[]"

	parser = rfc5424.NewParser()
)

// DHPLogMessage describes a structured log message from applications
type DHPLogMessage struct {
	Category            string          `json:"cat"`
	EventID             string          `json:"evt"`
	ApplicationVersion  string          `json:"ver"`
	Component           string          `json:"cmp"`
	ApplicationName     string          `json:"app"`
	ApplicationInstance string          `json:"inst"`
	ServerName          string          `json:"srv"`
	TransactionID       string          `json:"trns"`
	ServiceName         string          `json:"service"`
	LogTime             string          `json:"time"`
	OriginatingUser     string          `json:"usr"`
	Severity            string          `json:"sev"`
	LogData             DHPLogData      `json:"val"`
	Custom              json.RawMessage `json:"custom,omitempty"`
}

// DHPLogData represents the payload of DHPLogMessage
type DHPLogData struct {
	Message string `json:"message"`
}

// Logger implements logging interface
type Logger interface {
	Debugf(format string, args ...interface{})
}


// PHLogger implements all processing logic for parsing and forwarding logs
type PHLogger struct {
	debug        bool
	client       logging.Storer
	log          Logger
	buildVersion string
}

// NewPHLogger returns a new configured PHLogger instance
func NewPHLogger(storer logging.Storer, log Logger, buildVersion string) (*PHLogger, error) {
	var logger PHLogger

	logger.client = storer
	logger.log = log // Meta
	logger.buildVersion = buildVersion

	return &logger, nil
}


func BodyToResource(body []byte) (*logging.Resource, error) {
	syslogMessage, err := parser.Parse(body)
	if err != nil {
		return nil, err
	}
	if syslogMessage == nil || syslogMessage.Message() == nil {
		return nil, errNoMessage
	}
	resource, err := processMessage(syslogMessage)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

func (pl *PHLogger) flushBatch(buf *[]logging.Resource, count int) {
	fmt.Printf("Batch flushing %d messages\n", count)
	_, err := pl.client.StoreResources(*buf, count)
	if err != nil {
		fmt.Printf("Batch sending failed: %v\n", err)
	}
}

// ResourceWorker implements the worker process for parsing the queues
func (pl *PHLogger) ResourceWorker(resourceChannel <-chan logging.Resource, done <-chan bool) {
	var count int
	var dropped int
	buf := make([]logging.Resource, batchSize)

	for {
		select {
		case resource := <-resourceChannel:
			buf[count] = resource
			count++
			if count == batchSize {
				pl.flushBatch(&buf, count)
				count = 0
			}
		case <-time.After(1 * time.Second):
			if count > 0 {
				pl.flushBatch(&buf, count)
				count = 0
			}
			if dropped > 0 {
				fmt.Printf("Dropped %d messages\n", dropped)
				dropped = 0
			}
		case <-done:
			fmt.Printf("Worker received done message...\n")
			return
		}
	}
}

func processMessage(rfcLogMessage syslog.Message) (*logging.Resource, error) {
	var dhp DHPLogMessage
	var msg logging.Resource
	var logMessage *string

	if rfcLogMessage == nil {
		return nil, fmt.Errorf("no message found in syslog entry")
	}
	logMessage = rfcLogMessage.Message()

	for _, i := range ignorePatterns {
		if i.MatchString(*logMessage) {
			return nil, nil
		}
	}

	if req := requestUsersAPIPattern.FindStringSubmatch(*logMessage); req != nil {
		msg = wrapResource(req[1], rfcLogMessage, "buildVersion")
		return &msg, nil
	}
	msg = wrapResource("logproxy-wrapped", rfcLogMessage, "buildVersion")
	err := json.Unmarshal([]byte(*logMessage), &dhp)
	if err == nil {
		if dhp.OriginatingUser != "" {
			msg.OriginatingUser = EncodeString(dhp.OriginatingUser, originatingUsersInvalidCharacters)
		}
		if dhp.TransactionID != "" {
			msg.TransactionID = dhp.TransactionID
		}
		if dhp.EventID != "" {
			msg.EventID = EncodeString(dhp.EventID, eventIDInvalidCharacters)
		}
		if dhp.LogData.Message != "" {
			msg.LogData.Message = dhp.LogData.Message
		}
		if dhp.ApplicationVersion != "" {
			msg.ApplicationVersion = EncodeString(dhp.ApplicationVersion, versionInvalidCharacters)
		}
		if dhp.ApplicationName != "" {
			msg.ApplicationName = EncodeString(dhp.ApplicationName, applicationNameInvalidCharacters)
		}
		if dhp.ServiceName != "" {
			msg.ServiceName = EncodeString(dhp.ServiceName, otherNameInvalidCharacters)
		}
		if dhp.ServerName != "" {
			msg.ServerName = EncodeString(dhp.ServerName, otherNameInvalidCharacters)
		}
		if dhp.Category != "" {
			msg.Category = EncodeString(dhp.Category, defaultInvalidCharacters)
		}
		if dhp.Component != "" {
			msg.Component = EncodeString(dhp.Component, defaultInvalidCharacters)
		}
		if dhp.Severity != "" {
			msg.Severity = EncodeString(dhp.Severity, defaultInvalidCharacters)
		}
		msg.Custom = dhp.Custom
	}
	return &msg, nil
}

// EncodeString encodes all characters from the characterstoEncode set
func EncodeString(s string, charactersToEncode string) string {
	var res = strings.Builder{}
	for _, char := range s {
		if strings.ContainsRune(charactersToEncode, char) {
			encodedCharacter := fmt.Sprintf("%%%X", int(char))
			res.WriteString(encodedCharacter)
		} else {
			res.WriteRune(char)
		}
	}
	return res.String()
}

func wrapResource(originatingUser string, msg syslog.Message, buildVersion string) logging.Resource {
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
	lm.ApplicationVersion = buildVersion

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
		} else {
			lm.ApplicationInstance = EncodeString(*procID, applicationInstanceInvalidCharacters)
		}
	}

	// LogData
	lm.LogData.Message = "no message identified"
	if m := msg.Message(); m != nil {
		lm.LogData.Message = *m
	}

	return lm
}
