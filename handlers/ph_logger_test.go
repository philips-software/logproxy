package handlers

import (
	"os"
	"testing"

	"github.com/influxdata/go-syslog/rfc5424"
)

type NilLogger struct {
}

func (n *NilLogger) Debugf(format string, args ...interface{}) {
	// Don't log anything
}

func TestProcessMessage(t *testing.T) {
	os.Setenv("HSDP_LOGINGESTOR_KEY", "SharedKey")
	os.Setenv("HSDP_LOGINGESTOR_SECRET", "SharedSecret")
	os.Setenv("HSDP_LOGINGESTOR_URL", "https://foo")
	os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", "ProductKey")

	var payload = `Starting Application on 50676a99-dce0-418a-6b25-1e3d with PID 8 (/home/vcap/app/BOOT-INF/classes started by vcap in /home/vcap/app)`
	var appVersion = `1.0-f53a57a`
	var transactionID = `eea9f72c-09b6-4d56-905b-b518fc4dc5b7`

	var rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging 7215cbaa-464d-4856-967c-fd839b0ff7b2 [APP/PROC/WEB/0] - - {"app":"msa-eustaging","val":{"message":"` + payload + `"},"ver":"` + appVersion + `","evt":null,"sev":"INFO","cmp":"CPH","trns":"` + transactionID + `","usr":null,"srv":"msa-eustaging.eu-west.philips-healthsuite.com","service":"msa","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"Tracelog","time":"2018-09-07T15:39:21Z"}`

	var appName = `7215cbaa-464d-4856-967c-fd839b0ff7b2`
	var hostName = `suite-phs.staging.msa-eustaging`
	var nonDHPMessage = `<14>1 2018-09-07T15:39:18.517077+00:00 ` + hostName + ` ` + appName + ` [CELL/0] - - Starting health monitoring of container`

	parser := rfc5424.NewParser()

	phLogger, err := NewPHLogger(&NilLogger{})

	if err != nil {
		t.Fatalf("Expected NewPHLogger to succeed, got: %v\n", err)
	}
	msg, err := parser.Parse([]byte(rawMessage), nil)
	if err != nil {
		t.Fatalf("Expected Parse() to succeed, got: %v\n", err)
	}

	resource, err := phLogger.processMessage(msg)
	if err != nil {
		t.Fatalf("Expected processMessage() to succeed, got: %v\n", err)
	}
	if resource == nil {
		t.Errorf("Processed resource should not be nil")
	}
	if resource.ApplicationVersion != appVersion {
		t.Errorf("Expected ApplicationVersion to be `%s`, was `%s`", appVersion, resource.ApplicationVersion)
	}
	if resource.TransactionID != transactionID {
		t.Errorf("Expected TransactionID to be `%s`, was `%s`", transactionID, resource.TransactionID)
	}
	if resource.LogData.Message != payload {
		t.Errorf("Expected Message to be `%s`, was `%s`", payload, resource.LogData.Message)
	}

	msg, err = parser.Parse([]byte(nonDHPMessage), nil)
	if err != nil {
		t.Fatalf("Expected Parse() to succeed, got: %v\n", err)
	}
	resource, err = phLogger.processMessage(msg)
	if err != nil {
		t.Fatalf("Expected Parse() to succeed, got: %v\n", err)
	}
	if resource.LogTime != "2018-09-07T15:39:18.517Z" {
		t.Errorf("Unexpected LogTime: %s", resource.LogTime)
	}

	if resource.ApplicationName != appName {
		t.Errorf("Expected ApplicationName to be `%s`, was `%s`", appName, resource.ApplicationName)
	}
	if resource.ServerName != hostName {
		t.Errorf("Expected ApplicationName to be `%s`, was `%s`", hostName, resource.ServerName)
	}
}

func TestWrapResource(t *testing.T) {
	os.Setenv("HSDP_LOGINGESTOR_KEY", "SharedKey")
	os.Setenv("HSDP_LOGINGESTOR_SECRET", "SharedSecret")
	os.Setenv("HSDP_LOGINGESTOR_URL", "https://foo")
	os.Setenv("HSDP_LOGINGESTOR_PRODUCT_KEY", "ProductKey")

	var rtrLog = `<14>1 2019-04-12T19:34:43.530045+00:00 suite-xxx.staging.mps 042cbd0f-1a0e-4f77-ae39-a5c6c9fe2af9 [RTR/6] - - mps.domain.com - [2019-04-12T19:34:43.528+0000] "GET /test/bogus HTTP/1.1" 200 0 60 "-" "Not A Health Check" "10.10.66.246:48666" "10.10.17.45:61014" x_forwarded_for:"16.19.148.81, 10.10.66.246" x_forwarded_proto:"https" vcap_request_id:"77350158-4a69-47d6-731b-1bc0678db78d" response_time:0.001628089 app_id:"042cbd0f-1a0e-4f77-ae39-a5c6c9fe2af9" app_index:"0" x_b3_traceid:"6aa3915b88798203" x_b3_spanid:"6aa3915b88798203" x_b3_parentspanid:"-"`

	parser := rfc5424.NewParser()

	phLogger, err := NewPHLogger(&NilLogger{})

	if err != nil {
		t.Fatalf("Expected NewPHLogger to succeed, got: %v\n", err)
	}
	msg, err := parser.Parse([]byte(rtrLog), nil)
	if err != nil {
		t.Fatalf("Expected Parse() to succeed, got: %v\n", err)
	}

	resource, err := phLogger.processMessage(msg)
	if err != nil {
		t.Fatalf("Expected Parse() to succeed, got: %v\n", err)
	}
	if resource.LogTime != "2019-04-12T19:34:43.528Z" {
		t.Errorf("Unexpected LogTime: %s", resource.LogTime)
	}
}
