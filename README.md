# Logproxy

A microservice which acts as a logdrain and forwards messages to HSP Foundation logging. Supports the new v2 single tenant solution.

## Features

- Cloud foundry logdrain endpoint
- IronIO project logging endpoint 
- Batch uploads messages (max 25) for good performance
- Very lean, runs in just 32MB RAM
- [Plugin support](https://github.com/philips-software/logproxy-plugins/)
- Filter only mode
- OpenTracing support
- IAM Service Identity support

## Distribution

Logproxy is distributed as a [Docker image](https://github.com/philips-software/logproxy/pkgs/container/logproxy):

```shell
docker pull philipssoftware/logproxy
```

## Dependencies

By default Logproxy uses RabbitMQ for log buffering. This is useful for handlingspikes in log volume. You can also choose to use an internal Go `channel` based queue.

## Environment variables

| Variable                  | Description                          | Required            | Default |
|---------------------------|--------------------------------------|---------------------|---------|
| TOKEN                     | Token to use as part of logdrain URL | Yes                 |         |
| HSDP\_LOGINGESTOR\_PRODUCT\_KEY | Product key for v2 logging     | Yes (hsdp delivery) |         |
| LOGPROXY\_SYSLOG          | Enable or disable Syslog drain       |  No                 | true    |
| LOGPROXY\_IRONIO          | Enable or disable IronIO drain       |  No                 | false   |
| LOGPROXY\_QUEUE           | Use specific queue (rabbitmq, channel) | No                | rabbitmq |
| LOGPROXY\_PLUGINDIR       | Search for plugins in this directory | No                  |         |
| LOGPROXY\_DELIVERY        | Select delivery type (hsdp, none, buffer)    | No                  | hsdp    |
| LOGPROXY\_TRANSPORT\_URL  | The Jaeager transport endpoint       | No                  |         |

### IAM Service Identity based authentication (recommended)

| Variable                        | Description          | Required            | Default       |
|---------------------------------|----------------------|---------------------|---------------|
| LOGPROXY\_SERVICE\_ID           | IAM Service ID       | Yes (hsdp delivery) |               |
| LOGPROXY\_SERVICE\_PRIVATE\_KEY | IAM Service Private Key | Yes (hsdp delivery) |               |
| LOGPROXY\_REGION                | IAM Region           | Yes (hsdp delivery) | `us-east`     |
| LOGPROXY\_ENV                   | IAM Environment      | Yes (hsdp delivery) | `cllient-test` |

### API Signing based authentication

| Variable                  | Description                          | Required            | Default |
|---------------------------|--------------------------------------|---------------------|---------|
| HSDP\_LOGINGESTOR\_KEY    | HSDP logging service Key             | Yes (hsdp delivery) |         |
| HSDP\_LOGINGESTOR\_SECRET | HSDP logging service Secret          | Yes (hsdp delivery) |         |
| HSDP\_LOGINGESTOR\_URL    | HSPD logging service endpoint        | Yes (hsdp delivery) |         |


## Building

### Requirements

- [Go 1.16 or newer](https://golang.org/doc/install)

### Compiling

Clone the repo somewhere (preferably outside your GOPATH):

```
$ git clone https://github.com/philips-software/logproxy.git
$ cd logproxy
$ docker build .
```

## Installation

See the below manifest.yml file as an example.

```
applications:
- name: logproxy
  domain: your-domain.com
  docker:
    image: philipssoftware/logproxy:latest
  instances: 2
  memory: 64M
  disk_quota: 512M
  routes:
  - route: logproxy.your-domain.com
  env:
    HSDP_LOGINGESTOR_KEY: SomeKey
    HSDP_LOGINGESTOR_SECRET: SomeSecret
    HSDP_LOGINGESTOR_URL: https://logingestor-int2.us-east.philips-healthsuite.com
    HSDP_LOGINGESTOR_PRODUCT_KEY: product-uuid-here
    TOKEN: RandomTokenHere
  services:
  - rabbitmq
  stack: cflinuxfs3
```

Push your application:

```
cf push
```

If everything went OK logproxy should now be reachable on https://logproxy.your-domain.com . The logdrain endpoint would then be:

```
https://logproxy.your-domain.com/syslog/drain/RandomTokenHere
```

## Configure logdrains

### Syslog

In each space where you have apps running for which you'd like to drain logs define a user defined service called `logproxy`:

```
cf cups logproxy -l https://logproxy.your-domain.com/syslog/drain/RandomTokenHere
```  

Then, bind this service to any app which should deliver their logs:

```
cf bind-service some-app logproxy
```

and restart the app to activate the logdrain:

```
cf restart some-app
```

Logs should now start flowing from your app all the way to HSDP logging infra through logproxy. You can use Kibana for log searching.

### Structured logs

Below is an example of an HSDP LogEvent resource type for reference

```json
{
  "resourceType": "LogEvent",
  "id": "7f4c85a8-e472-479f-b772-2916353d02a4",
  "applicationName": "OPS",
  "eventId": "110114",
  "category": "TRACELOG",
  "component": "TEST",
  "transactionId": "2abd7355-cbdd-43e1-b32a-43ec19cd98f0",
  "serviceName": "OPS",
  "applicationInstance": "INST‐00002",
  "applicationVersion": "1.0.0",
  "originatingUser": "SomeUsr",
  "serverName": "ops-dev.apps.internal",
  "logTime": "2017-01-31T08:00:00Z",
  "severity": "INFO",
  "logData": {
    "message": "Test message"
  },
  "custom": {
  		"key1": "val1",
  		"key2": { "innerkey": "innervalue" }
   }
}
```

## IronIO

The IronIO logdrain is available on this endpoint: `/ironio/drain/:token`

You can configure via the iron.io settings screen of your project:

![settings screen](resources/IronIO-settings.png)

### Field Mapping

Logproxy maps the IronIO field to Syslog fields as follows

| IronIO field      | Syslog field        | LogEvent field      |
|-------------------|---------------------|---------------------|
| task\_id          | ProcID              | applicationInstance |
| code\_name        | AppName             | applicationName     |
| project\_id       | Hostname            | serverName          |
| message           | Message             | logData.message     |

## Filter only mode

You may choose to operate Logproxy in Filter only mode. It will listen 
for messages on the logdrain endpoints, run these through any active
filter plugins and then discard instead of delivering them to HSDP logging.
This is useful if you are using plugins for real-time processing only.
To enable filter only mode set `LOGPROXY_DELIVERY` to `none`

```
...
env:
  LOGPROXY_DELIVERY: none
...
```

See the [Logproxy plugins](https://github.com/philips-software/logproxy-plugins) project for more details on plugins.

## TODO

- Better handling of HTTP 635 errors
