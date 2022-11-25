package queue

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go"
	"github.com/philips-software/logproxy/shared"

	"github.com/google/uuid"
	"github.com/influxdata/go-syslog/v2"
	"github.com/influxdata/go-syslog/v2/rfc5424"
	"github.com/philips-software/go-hsdp-api/logging"
)

var (
	batchSize     = 25
	rtrTimeFormat = "2006-01-02T15:04:05.000Z0700"
	vcapPattern   = regexp.MustCompile(`vcap_request_id:"(?P<requestID>[^"]+)"`)
	rtrPattern    = regexp.MustCompile(`\[RTR/(?P<index>\d+)]`)
	rtrFormat     = regexp.MustCompile(`(?P<hostname>[^?/\s]+) - \[(?P<time>[^/\s]+)]`)
	Base64Pattern = regexp.MustCompile(`^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$`)

	errNoMessage = errors.New("no message in syslogMessage")

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
	TraceID             string          `json:"trace"`
	SpanID              string          `json:"span"`
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

// Deliverer implements all processing logic for parsing and forwarding logs
type Deliverer struct {
	Debug        bool
	storer       logging.Storer
	log          Logger
	buildVersion string
	manager      *shared.PluginManager
	metrics      Metrics
}

// NewDeliverer returns a new configured Deliverer instance
func NewDeliverer(storer logging.Storer, log Logger, manager *shared.PluginManager, buildVersion string, metrics Metrics) (*Deliverer, error) {
	var logger Deliverer

	logger.storer = storer
	logger.log = log // Meta
	logger.buildVersion = buildVersion
	logger.manager = manager
	logger.metrics = metrics

	return &logger, nil
}

// BodyToResource takes the raw body and transforms it to a
// logging.Resource instance
func BodyToResource(body []byte, m Metrics) (*logging.Resource, error) {
	syslogMessage, err := parser.Parse(body)
	if err != nil {
		return nil, err
	}
	if syslogMessage == nil || syslogMessage.Message() == nil {
		return nil, errNoMessage
	}
	resource, err := ProcessMessage(syslogMessage, m)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

func (pl *Deliverer) flushBatch(ctx context.Context, resources []logging.Resource, count int, queue Queue) (int, error) {
	tracer := opentracing.GlobalTracer()
	span, _ := opentracing.StartSpanFromContextWithTracer(ctx, tracer, "deliverer_flush_batch")
	defer span.Finish()
	fmt.Printf("batch flushing %d messages\n", count)

	resp, err := pl.storer.StoreResources(resources, count)
	if err != nil { // Unpack and send individually
		if resp == nil {
			fmt.Printf("unexpected error for StoreResource(): %v\n", err)
			return count, err
		}
		// Unpack and send individual messages
		for i := 0; i < count; i++ {
			fmt.Printf("resending %d\n", i+1)
			_, err = pl.storer.StoreResources([]logging.Resource{resources[i]}, 1)
			if err != nil {
				_ = queue.DeadLetter(resources[i])
				fmt.Printf("permanent failure sending %d resource: [%v] error: %v\n", i+1, resources[i], err)
			}
		}
	}

	return count, nil
}

// ResourceWorker implements the worker process for parsing the queues
func (pl *Deliverer) ResourceWorker(queue Queue, done <-chan bool, _ *zipkin.Tracer) {
	var count int
	var totalStored int64
	buf := make([]logging.Resource, batchSize)
	resourceChannel := queue.Output()

	fmt.Printf("Starting ResourceWorker...\n")
	for {
		ctx := context.Background()
		select {
		case resource := <-resourceChannel:
			if resource.ApplicationVersion == "" {
				resource.ApplicationVersion = pl.buildVersion
			}
			if drop := pl.processFilters(ctx, &resource); drop {
				continue
			}
			buf[count] = resource
			count++
			if count == batchSize {
				stored, _ := pl.flushBatch(ctx, buf, count, queue)
				totalStored += int64(stored)
				count = 0
			}
		case <-time.After(500 * time.Millisecond):
			if count > 0 {
				stored, _ := pl.flushBatch(ctx, buf, count, queue)
				totalStored += int64(stored)
				count = 0
			}
		case <-done:
			if count > 0 {
				stored, _ := pl.flushBatch(ctx, buf, count, queue)
				totalStored += int64(stored)
			}
			fmt.Printf("Worker received done message...%d stored\n", totalStored)
			return
		}
	}
}

func (pl *Deliverer) processFilters(ctx context.Context, resource *logging.Resource) bool {
	tracer := opentracing.GlobalTracer()
	span, _ := opentracing.StartSpanFromContextWithTracer(ctx, tracer, "process_filter")
	defer span.Finish()
	if pl.manager == nil {
		return false
	}
	for _, p := range pl.manager.Plugins() {
		r, drop, modified, _ := p.App.Filter(*resource)
		if drop {
			if pl.metrics != nil {
				pl.metrics.IncPluginDropped()
			}
			return true
		}
		if modified {
			if pl.metrics != nil {
				pl.metrics.IncPluginModified()
			}
			*resource = r
		}
	}
	return false
}

func ProcessMessage(rfcLogMessage syslog.Message, m Metrics) (*logging.Resource, error) {
	var dhp DHPLogMessage
	var msg logging.Resource
	var logMessage *string

	if rfcLogMessage == nil {
		return nil, fmt.Errorf("no message found in syslog entry")
	}
	logMessage = rfcLogMessage.Message()

	err := json.Unmarshal([]byte(*logMessage), &msg)
	if err == nil && msg.ResourceType == "LogEvent" {
		if !Base64Pattern.MatchString(msg.LogData.Message) { // Encode
			msg.LogData.Message = base64.StdEncoding.EncodeToString([]byte(msg.LogData.Message))
			m.IncEnhancedEncodedMessage()
		}
		if msg.TransactionID == "" { // Generate missing transactionId
			msg.TransactionID = uuid.NewString()
			m.IncEnhancedTransactionID()
		}
		if !msg.Valid() {
			return nil, msg.Error
		}
		return &msg, nil
	}
	if err != nil {
		fmt.Printf("wrapping message: %v '%s'\n", err, *logMessage)
	}

	msg = wrapResource("logproxy-wrapped", rfcLogMessage)

	err = json.Unmarshal([]byte(*logMessage), &dhp)
	if err == nil {
		if dhp.TransactionID != "" {
			msg.TransactionID = dhp.TransactionID
		}
		if dhp.LogData.Message != "" {
			if !Base64Pattern.MatchString(dhp.LogData.Message) { // Encode
				msg.LogData.Message = base64.StdEncoding.EncodeToString([]byte(dhp.LogData.Message))
				m.IncEnhancedEncodedMessage()
			} else { // Already base64 encoded
				msg.LogData.Message = dhp.LogData.Message
			}
		}
		if dhp.ApplicationInstance != "" {
			msg.ApplicationInstance = dhp.ApplicationInstance
		}
		var encoderTasks = []struct {
			src          string
			dest         *string
			invalidChars string
		}{
			{dhp.OriginatingUser, &msg.OriginatingUser, originatingUsersInvalidCharacters},
			{dhp.EventID, &msg.EventID, eventIDInvalidCharacters},
			{dhp.ApplicationVersion, &msg.ApplicationVersion, versionInvalidCharacters},
			{dhp.ApplicationName, &msg.ApplicationName, applicationNameInvalidCharacters},
			{dhp.ServiceName, &msg.ServiceName, otherNameInvalidCharacters},
			{dhp.ServerName, &msg.ServerName, otherNameInvalidCharacters},
			{dhp.Category, &msg.Category, defaultInvalidCharacters},
			{dhp.Component, &msg.Component, defaultInvalidCharacters},
			{dhp.Severity, &msg.Severity, defaultInvalidCharacters},
			{dhp.TraceID, &msg.TraceID, defaultInvalidCharacters},
			{dhp.SpanID, &msg.SpanID, defaultInvalidCharacters},
		}
		for _, task := range encoderTasks {
			if task.src != "" {
				*task.dest = EncodeString(task.src, task.invalidChars)
			}
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

func wrapResource(originatingUser string, msg syslog.Message) logging.Resource {
	var lm logging.Resource

	// ID
	lm.ID = uuid.NewString()

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
		lm.TransactionID = uuid.NewString()
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

	// OriginatingUser
	lm.OriginatingUser = originatingUser

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
	lm.LogTime = time.Now().Format(logging.TimeFormat)
	if m := msg.Timestamp(); m != nil {
		lm.LogTime = m.Format(logging.TimeFormat)
	}
	parseProcID(msg, &lm)

	// LogData
	lm.LogData.Message = "no message identified"
	if m := msg.Message(); m != nil {
		lm.LogData.Message = *m
	}

	lm.ResourceType = "LogEvent"
	// Base64 encode
	lm.LogData.Message = base64.StdEncoding.EncodeToString([]byte(lm.LogData.Message))

	return lm
}

func parseProcID(msg syslog.Message, lm *logging.Resource) {
	if procID := msg.ProcID(); procID != nil {
		if rtrPattern.FindStringSubmatch(*procID) != nil && msg.Message() != nil {
			m := msg.Message()
			if rtr := rtrFormat.FindStringSubmatch(*m); rtr != nil {
				if rtrTime, err := time.Parse(rtrTimeFormat, rtr[2]); err == nil {
					lm.LogTime = rtrTime.Format(logging.TimeFormat)
				}
			}
		} else {
			lm.ApplicationInstance = EncodeString(*procID, applicationInstanceInvalidCharacters)
		}
	}
}
