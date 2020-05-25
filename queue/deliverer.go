package queue

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
	rtrTimeFormat  = "2006-01-02T15:04:05.000Z0700"
	ignorePatterns = []*regexp.Regexp{
		regexp.MustCompile(`Consul Health Check`),
		regexp.MustCompile(`POST /syslog/drain`),
		regexp.MustCompile(`GET /api/version`),
	}
	requestUsersAPIPattern = regexp.MustCompile(`/api/users/(?P<userID>[^?/\s]+)`)
	vcapPattern            = regexp.MustCompile(`vcap_request_id:"(?P<requestID>[^"]+)"`)
	rtrPattern             = regexp.MustCompile(`\[RTR/(?P<index>\d+)]`)
	rtrFormat              = regexp.MustCompile(`(?P<hostname>[^?/\s]+) - \[(?P<time>[^/\s]+)]`)

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
	debug        bool
	storer       logging.Storer
	log          Logger
	buildVersion string
}

// NewDeliverer returns a new configured Deliverer instance
func NewDeliverer(storer logging.Storer, log Logger, buildVersion string) (*Deliverer, error) {
	var logger Deliverer

	logger.storer = storer
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

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (pl *Deliverer) flushBatch(resources []logging.Resource, count int, queue Queue) (int, error) {
	fmt.Printf("Batch flushing %d messages\n", count)
	maxLoop := count
	l := 0

	for {
		l++
		resp, err := pl.storer.StoreResources(resources, count)
		if err == nil { // Happy flow
			break
		}
		if resp == nil {
			fmt.Printf("Unexpected error for StoreResource(): %v\n", err)
			continue
		}
		nrErrors := len(resp.Failed)
		keys := make([]int, 0, nrErrors)
		for k := range resp.Failed {
			keys = append(keys, k)
		}
		// Remove offending messages and resend
		pos := 0
		for i := 0; i < count; i++ {
			if contains(keys, i) {
				_ = queue.DeadLetter(resources[i])
				continue
			}
			resources[pos] = resources[i]
			pos++
		}
		count = pos
		fmt.Printf("Found %d errors. Resending %d\n", nrErrors, count)

		if l > maxLoop || count <= 0 {
			fmt.Printf("Maximum retries reached or nothingt to send. Bailing..\n")
			break
		}
	}
	return count, nil
}

// ResourceWorker implements the worker process for parsing the queues
func (pl *Deliverer) ResourceWorker(queue Queue, done <-chan bool) {
	var count int
	var totalStored int64
	buf := make([]logging.Resource, batchSize)
	resourceChannel := queue.Output()

	fmt.Printf("Starting ResourceWorker...\n")
	for {
		select {
		case resource := <-resourceChannel:
			buf[count] = resource
			count++
			if count == batchSize {
				stored, _ := pl.flushBatch(buf, count, queue)
				totalStored += int64(stored)
				count = 0
			}
		case <-time.After(500 * time.Millisecond):
			if count > 0 {
				stored, _ := pl.flushBatch(buf, count, queue)
				totalStored += int64(stored)
				count = 0
			}
		case <-done:
			if count > 0 {
				stored, _ := pl.flushBatch(buf, count, queue)
				totalStored += int64(stored)
			}
			fmt.Printf("Worker received done message...%d stored\n", totalStored)
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
		if dhp.TransactionID != "" {
			msg.TransactionID = dhp.TransactionID
		}
		if dhp.LogData.Message != "" {
			msg.LogData.Message = dhp.LogData.Message
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
		uuidV4, _ := uuid.V4()
		lm.TransactionID = uuidV4.String()
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
