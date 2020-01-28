package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/influxdata/go-syslog/v2/rfc5424"
	"github.com/labstack/echo"
	"github.com/loafoe/go-rabbitmq"
	"github.com/streadway/amqp"
)

var (
	ironIOPayloadRegex = regexp.MustCompile(`severity=(?P<severity>[^\?,]+), task_id: (?P<taskID>[^\?,]+), code_name: (?P<codeName>[^\?,]+), project_id: (?P<projectID>[^\?\s]+) -- (?P<body>.*)`)
)

type IronIOHandler struct {
	producer rabbitmq.Producer
	debug    bool
	token    string
}

func NewIronIOHandler(token string, producer rabbitmq.Producer) (*IronIOHandler, error) {
	if token == "" {
		return nil, fmt.Errorf("Missing TOKEN value")
	}
	handler := &IronIOHandler{}
	handler.token = token
	handler.producer = producer

	if os.Getenv("DEBUG") == "true" {
		handler.debug = true
	}
	return handler, nil
}

func ironToRFC5424(now time.Time, ironString string) string {
	msg := &rfc5424.SyslogMessage{}

	msg.SetPriority(14)
	msg.SetVersion(1)
	msg.SetTimestamp(now.Format(logTimeFormat))

	match := ironIOPayloadRegex.FindStringSubmatch(ironString)
	if match != nil {
		if len(match) == 6 {
			msg.SetProcID(match[2])
			msg.SetAppname(match[3])
			msg.SetHostname(match[4])
			msg.SetMessage(match[5])
		} else {
			msg.SetMessage("mismatch: " + ironString)
		}
	} else {
		msg.SetMessage("nomatch: " + ironString) // Naive
	}

	out, _ := msg.String()
	return out
}

func (h *IronIOHandler) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		t := c.Param("token")
		if h.token != t {
			return c.String(http.StatusUnauthorized, "")
		}
		b, _ := ioutil.ReadAll(c.Request().Body)
		go h.push([]byte(ironToRFC5424(time.Now().UTC(), string(b))))
		return c.String(http.StatusOK, "")
	}
}

func (h *IronIOHandler) push(raw []byte) {
	err := h.producer.Publish(Exchange, RoutingKey, amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     "application/octet-stream",
		ContentEncoding: "",
		Body:            raw,
		DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
		Priority:        0,              // 0-9
		// a bunch of application/implementation-specific fields
	})
	if err != nil {
		fmt.Printf("Error publishing: %v\n", err)
	}
}
