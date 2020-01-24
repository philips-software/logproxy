package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/influxdata/go-syslog/v2/rfc5424"
	"github.com/labstack/echo"
	"github.com/loafoe/go-rabbitmq"
	"github.com/streadway/amqp"
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
	msg.SetTimestamp(now.Format(time.RFC3339))
	msg.SetMessage(ironString) // Naive first, we will parse it later

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
		go h.push([]byte(ironToRFC5424(time.Now(), string(b))))
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
