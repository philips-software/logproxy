package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo"
	"github.com/loafoe/go-rabbitmq"
	"github.com/streadway/amqp"
)

var (
	Exchange   = "logproxy"
	RoutingKey = "new.rfc5424"
)

type SyslogHandler struct {
	PHLogger *PHLogger
	producer *rabbitmq.Producer
	debug    bool
	token    string
}

func NewSyslogHandler(token string, log Logger) (*SyslogHandler, error) {
	var err error
	if token == "" {
		return nil, fmt.Errorf("Missing TOKEN value")
	}

	handler := &SyslogHandler{}
	handler.PHLogger, err = NewPHLogger(log)
	if err != nil {
		return nil, err
	}
	handler.token = token
	handler.producer, err = rabbitmq.NewProducer(rabbitmq.Config{
		Exchange:     Exchange,
		ExchangeType: "topic",
		Durable:      false,
	})
	if os.Getenv("DEBUG") == "true" {
		handler.debug = true
	}
	return handler, nil
}

func (h *SyslogHandler) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		t := c.Param("token")
		if h.token != t {
			c.String(http.StatusUnauthorized, "")
			return fmt.Errorf("Invalid token")
		}
		b, _ := ioutil.ReadAll(c.Request().Body)
		go h.push(b)
		c.String(http.StatusOK, "")
		return nil
	}
}

func (h *SyslogHandler) push(raw []byte) error {
	return h.producer.Publish(Exchange, RoutingKey, amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     "application/octet-stream",
		ContentEncoding: "",
		Body:            raw,
		DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
		Priority:        0,              // 0-9
		// a bunch of application/implementation-specific fields
	})

}
