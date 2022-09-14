package queue_test

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/philips-software/logproxy/queue"

	"github.com/influxdata/go-syslog/v2/rfc5424"
	"github.com/stretchr/testify/assert"
)

var testBuild = "v0.0.0-test"

func TestCustomJSONInProcessMessage(t *testing.T) {
	customJSON := `{"app":"mbcs-dev","val":{"message":"Log message"},"ver":"2.0-2fe99a7","evt":null,"sev":"INFO","cmp":"CPH","trns":"f9bbda22-1498-4096-7c3a-ded96eedf79d","usr":"63a49d36-4d18-4651-a4b2-2116fa8037fa","srv":"mbcs-dev.apps.internal","service":"mbcs","inst":"a2b1bb56-0467-47bf-41fc-8118","cat":"Tracelog","time":"2020-01-25T20:10:34Z"}`
	rawMessage := "<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.dev.msa-dev appName [APP/PROC/WEB/0] - - " + customJSON

	parser := rfc5424.NewParser()

	deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	if !assert.Nilf(t, err, "Expected NewDeliverer() to succeed") {
		return
	}
	if !assert.NotNil(t, deliverer) {
		return
	}
	deliverer.Debug = true
	msg, err := parser.Parse([]byte(rawMessage))
	assert.Nilf(t, err, "Expected Parse() to succeed")
	resource, err := queue.ProcessMessage(msg, &nilMetrics{})
	assert.Nilf(t, err, "Expected ProcessMessage() to succeed")
	assert.Equal(t, `Log message`, resource.LogData.Message)

}

func TestDeliveryToResource(t *testing.T) {
	const rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging appName [APP/PROC/WEB/0] - - {"app":"appName","val":{"message":"bericht"},"ver":"1.0.0","evt":"eventID","sev":"info","cmp":"component","trns":"transactionID","usr":null,"srv":"serverName","service":"serviceName","usr":"foo","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"xxx","time":"2018-09-07T15:39:21Z"}`

	r, err := queue.BodyToResource([]byte(rawMessage), &nilMetrics{})
	assert.Nil(t, err)
	assert.NotNil(t, r)
	// Empty body
	_, err = queue.BodyToResource([]byte(""), &nilMetrics{})
	assert.NotNil(t, err)
}

func TestProcessMessagePassthroughs(t *testing.T) {
	var rawMessage = `{"serviceName":"5c737703-fef3-49ed-9efe-51ba3a68c3e7","transactionId":"07bdcdd6-b398-48e2-99f9-6ed6209c84f2","applicationName":"5c737703-fef3-49ed-9efe-51ba3a68c3e7","applicationInstance":"%5BAPP/PROC/WEB/0%5D","resourceType":"LogEvent","severity":"informational","logTime":"2021-11-01T11:45:22.462Z","applicationVersion":"v1.2.2-deadbeaf","originatingUser":"logproxy-wrapped","logData":{"message":"Hello world"},"eventId":"1","component":"logproxy","id":"d6c0adef-54c0-4452-ad71-65b01eb1becd","category":"ApplicationLog","custom":{"tenantId":"265a6faa-c3be-4b5a-b827-f8a8bc569064","timestampUtc":"2021-11-01T11:45:22.430Z","userId":"foo@bar.com","durationMs":"3000"},"serverName":"client-xxx.dev.userpreferenceservice"}`
	var payload = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging appname [APP/PROC/WEB/0] - - ` + rawMessage

	parser := rfc5424.NewParser()

	deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	deliverer.Debug = true

	msg, err := parser.Parse([]byte(payload))
	if !assert.Nilf(t, err, "Expected Parse() to succeed") {
		return
	}
	resource, err := queue.ProcessMessage(msg, &nilMetrics{})
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, resource) {
		return
	}
	assert.Equal(t, "LogEvent", resource.ResourceType)
}

func TestProcessMessage(t *testing.T) {
	const payload = "Starting Application on 50676a99-dce0-418a-6b25-1e3d with PID 8 (/home/vcap/app/BOOT-INF/classes started by vcap in /home/vcap/app)"
	const appVersion = "1.0-f53a57a>"
	const transactionID = "eea9f72c-09b6-4d56-905b-b518fc4dc5b7"
	const originatingUser = "logproxy$,"

	const appName = "7215cbaa-464d-4856-967c-fd839b0ff7b2#"
	const appInstance = "0485989d-b532-4c8b-b520-9a2b598df630"
	const logAppName = "TestAppName#"
	const eventTracingID = "b4b0b7d089591aa5:b4b0b7d089591aa5"
	const severity = "FATAL|"
	const component = "TestComponent,,"
	const serviceName = "com.philips.MyLoggingClass()"
	const serverName = "@396f1a94-86f3-470b-784c-17cc=="
	// No invalid characters in category to test the encode doesn't modify the string when not needed
	const category = "TraceLog"
	const rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging ` + appName + ` [APP/PROC/WEB/0] - - {"app":"` + logAppName + `","val":{"message":"` + payload + `"},"ver":"` + appVersion + `","evt":"` + eventTracingID + `","sev":"` + severity + `","cmp":"` + component + `","trns":"` + transactionID + `","usr":null,"srv":"` + serverName + `","service":"` + serviceName + `","usr":"` + originatingUser + `","inst":"` + appInstance + `","cat":"` + category + `","time":"2018-09-07T15:39:21Z"}`

	const hostName = `suite-phs.staging.msa-eustaging`
	const nonDHPMessage = `<14>1 2018-09-07T15:39:18.517077+00:00 ` + hostName + ` ` + appName + ` [CELL/0] - - Starting health monitoring of container`

	parser := rfc5424.NewParser()

	deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	deliverer.Debug = true

	msg, err := parser.Parse([]byte(rawMessage))
	assert.Nilf(t, err, "Expected Parse() to succeed")

	resource, err := queue.ProcessMessage(msg, &nilMetrics{})
	assert.Nilf(t, err, "Expected ProcessMessage() to succeed")
	assert.NotNilf(t, resource, "Processed resource should not be nil")
	assert.Equal(t, "1.0-f53a57a%3E", resource.ApplicationVersion)
	assert.Equal(t, "TestAppName%23", resource.ApplicationName)
	assert.Equal(t, category, resource.Category)
	assert.Equal(t, "TestComponent%2C%2C", resource.Component)
	assert.Equal(t, transactionID, resource.TransactionID)
	assert.Equal(t, appInstance, resource.ApplicationInstance)
	assert.Equal(t, "b4b0b7d089591aa5%3Ab4b0b7d089591aa5", resource.EventID)
	assert.Equal(t, "com.philips.MyLoggingClass%28%29", resource.ServiceName)
	assert.Equal(t, "%40396f1a94-86f3-470b-784c-17cc%3D%3D", resource.ServerName)
	assert.Equal(t, "FATAL%7C", resource.Severity)
	assert.Equal(t, payload, resource.LogData.Message)
	assert.Equal(t, "logproxy%24,", resource.OriginatingUser)

	msg, err = parser.Parse([]byte(nonDHPMessage))

	assert.Nilf(t, err, "Expected Parse() to succeed")

	resource, err = queue.ProcessMessage(msg, &nilMetrics{})

	assert.Nilf(t, err, "Expected Parse() to succeed")

	assert.Equal(t, "2018-09-07T15:39:18.517Z", resource.LogTime)
	assert.Equal(t, appName, resource.ApplicationName)
	assert.Equal(t, hostName, resource.ServerName)
	assert.Equal(t, "U3RhcnRpbmcgaGVhbHRoIG1vbml0b3Jpbmcgb2YgY29udGFpbmVy", resource.LogData.Message)
}

func TestProcessMessageEncodingCheck(t *testing.T) {
	rawMessage := `{"resourceType":"LogEvent","id":"c183607a-04c9-4e4d-a37d-6022ca955ac5","transactionId":"a7bd5c3b-e5da-4238-8655-8fcd01f64e7a","logTime":"2022-09-14T09:01:42.278Z","category":"TraceLog","eventId":"0","component":"WeatherForecastController","traceId":"b98a648c7047f1cc1d0b680515597ef0","spanId":"9fafeb189b8288d5","originatingUser":"User","logData":{"message":"I am logging sample e122902e-9e46-4665-ad07-4d51f36c8353"},"severity":"Information","applicationName":"Testlogapp","applicationVersion":"93dc5412-c7cd-4e93-979c-a87e4d8550ae","serviceName":"client-test.testorg.testapp","serverName":"DefaultServer","applicationInstance":"deda3c55-4a2f-480e-92a7-286429b0fc48"}`
	encodedMessagePart := "SSBhbSBsb2dnaW5nIHNhbXBsZSBlMTIyOTAyZS05ZTQ2LTQ2NjUtYWQwNy00ZDUxZjM2YzgzNTM="
	var payload = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging appname [APP/PROC/WEB/0] - - ` + rawMessage

	parser := rfc5424.NewParser()

	deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	deliverer.Debug = true

	msg, err := parser.Parse([]byte(payload))
	if !assert.Nilf(t, err, "Expected Parse() to succeed") {
		return
	}
	resource, err := queue.ProcessMessage(msg, &nilMetrics{})
	if !assert.Nil(t, err) {
		return
	}
	if !assert.NotNil(t, resource) {
		return
	}
	assert.Equal(t, "LogEvent", resource.ResourceType)
	assert.Equal(t, encodedMessagePart, resource.LogData.Message)
	assert.True(t, queue.Base64Pattern.MatchString(encodedMessagePart))
}

func TestResourceWorker(t *testing.T) {
	const payload = `Starting Application on 50676a99-dce0-418a-6b25-1e3d with PID 8 (/home/vcap/app/BOOT-INF/classes started by vcap in /home/vcap/app)`
	const appVersion = `1.0-f53a57a`
	const transactionID = `eea9f72c-09b6-4d56-905b-b518fc4dc5b7`
	const rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging 7215cbaa-464d-4856-967c-fd839b0ff7b2 [APP/PROC/WEB/0] - - {"app":"msa-eustaging","val":{"message":"` + payload + `"},"ver":"` + appVersion + `","evt":null,"sev":"INFO","cmp":"CPH","trns":"` + transactionID + `","usr":null,"srv":"msa-eustaging.eu-west.philips-healthsuite.com","service":"msa","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"Tracelog","time":"2018-09-07T15:39:21Z"}`

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	deliverer.Debug = true

	q, _ := queue.NewChannelQueue()
	done, _ := q.Start()

	go deliverer.ResourceWorker(q, done, nil)

	go func() {
		for i := 0; i < 25; i++ {
			_ = q.Push([]byte(rawMessage))
		}
	}()

	time.Sleep(600 * time.Millisecond) // Wait for the flush to happen

	done <- true

	os.Stdout = old
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	assert.Regexp(t, regexp.MustCompile("batch flushing 25 messages"), buf.String())
}

func TestWrapResource(t *testing.T) {
	const rtrLog = `<14>1 2019-04-12T19:34:43.530045+00:00 suite-xxx.staging.mps 042cbd0f-1a0e-4f77-ae39-a5c6c9fe2af9 [RTR/6] - - mps.domain.com - [2019-04-12T19:34:43.528+0000] "GET /test/bogus HTTP/1.1" 200 0 60 "-" "Not A Health Check" "10.10.66.246:48666" "10.10.17.45:61014" x_forwarded_for:"16.19.148.81, 10.10.66.246" x_forwarded_proto:"https" vcap_request_id:"77350158-4a69-47d6-731b-1bc0678db78d" response_time:0.001628089 app_id:"042cbd0f-1a0e-4f77-ae39-a5c6c9fe2af9" app_index:"0" x_b3_traceid:"6aa3915b88798203" x_b3_spanid:"6aa3915b88798203" x_b3_parentspanid:"-"`

	parser := rfc5424.NewParser()

	Deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	Deliverer.Debug = true

	msg, err := parser.Parse([]byte(rtrLog))
	assert.Nilf(t, err, "Expected Parse() to succeed")

	resource, err := queue.ProcessMessage(msg, &nilMetrics{})

	assert.Nilf(t, err, "Expected ProcessMessage() to succeed")
	assert.Equal(t, "2019-04-12T19:34:43.528Z", resource.LogTime)
}

func TestDroppedMessages(t *testing.T) {
	const payload = `Starting Application on 50676a99-dce0-418a-6b25-1e3d with PID 8 (/home/vcap/app/BOOT-INF/classes started by vcap in /home/vcap/app)`
	const appVersion = `1.0-f53a57a`
	const transactionID = `eea9f72c-09b6-4d56-905b-b518fc4dc5b7`
	const rawMessage = `<14>1 2018-09-07T15:39:21.132433+00:00 suite-phs.staging.msa-eustaging 7215cbaa-464d-4856-967c-fd839b0ff7b2 [APP/PROC/WEB/0] - - {"app":"msa-eustaging","val":{"message":"` + payload + `"},"ver":"` + appVersion + `","evt":null,"sev":"INFO","cmp":"CPH","trns":"` + transactionID + `","usr":null,"srv":"msa-eustaging.eu-west.philips-healthsuite.com","service":"msa","inst":"50676a99-dce0-418a-6b25-1e3d","cat":"Tracelog","time":"2018-09-07T15:39:21Z"}`

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	deliverer.Debug = true

	q, _ := queue.NewChannelQueue()
	done, _ := q.Start()

	go deliverer.ResourceWorker(q, done, nil)

	go func() {
		for i := 0; i < 23; i++ {

			_ = q.Push([]byte(rawMessage))
		}
	}()

	time.Sleep(600 * time.Millisecond) // Wait for the flush to happen

	done <- true

	os.Stdout = old
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	assert.Regexp(t, regexp.MustCompile("batch flushing 23 messages"), buf.String())
	assert.Regexp(t, regexp.MustCompile("Found 2 errors. Resending 21"), buf.String())
}

func TestEncodeString(t *testing.T) {
	assert.Equal(t, "%24%26%2B%2C%3A%3B%3D%3F%40%23%7C%3C%3E%28%29%5B%5D", queue.EncodeString("$&+,:;=?@#|<>()[]", "$&+,:;=?@#|<>()[]"))
	assert.Equal(t, "$&+,:;=?@#|<>()[]", queue.EncodeString("$&+,:;=?@#|<>()[]", ""))
	assert.Equal(t, "abc", queue.EncodeString("abc", ""))
}

func TestUserMessage(t *testing.T) {
	const userLog = `<14>1 2019-04-12T19:34:43.530045+00:00 suite-xxx.staging.mps 042cbd0f-1a0e-4f77-ae39-a5c6c9fe2af9 [RTR/6] - - mps.domain.com - [2019-04-12T19:34:43.528+0000] "GET /api/users/originating-user/bogus HTTP/1.1" 200 0 60 "-" "Foo Check" "10.10.66.246:48666" "10.10.17.45:61014" x_forwarded_for:"16.19.148.81, 10.10.66.246" x_forwarded_proto:"https" vcap_request_id:"77350158-4a69-47d6-731b-1bc0678db78d" response_time:0.001628089 app_id:"042cbd0f-1a0e-4f77-ae39-a5c6c9fe2af9" app_index:"0" x_b3_traceid:"6aa3915b88798203" x_b3_spanid:"6aa3915b88798203" x_b3_parentspanid:"-"`

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Deliverer, err := queue.NewDeliverer(&nilStorer{}, &nilLogger{}, nil, testBuild, &nilMetrics{})
	assert.Nilf(t, err, "Expected NewDeliverer() to succeed")
	Deliverer.Debug = true

	q, _ := queue.NewChannelQueue()
	done, _ := q.Start()

	go Deliverer.ResourceWorker(q, done, nil)

	_ = q.Push([]byte(userLog))

	time.Sleep(600 * time.Millisecond) // Wait for the flush to happen

	done <- true

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	assert.Regexp(t, regexp.MustCompile("batch flushing 1 messages"), buf.String())
}
