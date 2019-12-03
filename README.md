# Logproxy
[![Build Status](https://dev.azure.com/philips-software/logproxy/_apis/build/status/philips-software.logproxy?branchName=master)](https://dev.azure.com/philips-software/logproxy/_build/latest?definitionId=2&branchName=master)

A microservice which acts as a logdrain and forwards messages to HSDP Foundation logging. Supports the new HSDP v2 single tenant solution.

# Features
- Supports v2 of the HSDP logging API
- Batch uploads messages (max 25) for good performance
- Very lean (64MB RAM)

# Dependencies
A RabbitMQ instance is required. This is used to handle spikes in log volume.

# Environment variables

| Variable                  | Description                          | Required |
|---------------------------|--------------------------------------|----------|
| TOKEN                     | Token to use as part of logdrain URL | Yes      |
| HSDP\_LOGINGESTOR\_KEY    | HSDP logging service Key             | Yes      |
| HSDP\_LOGINGESTOR\_SECRET | HSDP logging service Secret          | Yes      |
| HSDP\_LOGINGESTOR\_URL    | HSPD logging service endpoint        | Yes      |
| HSDP\_LOGINGESTOR\_PRODUCT\_KEY | Product key for v2 logging     | Yes      |

# Building

## Requirements

- [Go 1.12 or newer](https://golang.org/doc/install)

## Compiling

Clone the repo somewhere (preferably outside your GOPATH):

```
$ git clone https://github.com/philips-software/logproxy.git
$ cd logproxy
$ ./buildscript.sh
```

This produce a `logproxy` binary executable in the `build` directory ready for use. The output also contains the unit test coverage, unit test results and a JUnit compatible format of the unit test execution result.

# Docker

Alternatively, you can use the included Dockerfile to build a docker image which can be deployed to CF directly.

```
$ git clone https://github.com/philips-software/logproxy.git
$ cd logproxy
$ docker build -t build -f Dockerfile.build .
$ docker run --name build --rm -v `pwd`:/src build
$ docker build -t logproxy -f Dockerfile.dist .
```

# Installation
See the below manifest.yml file as an example. Make sure you include the `logproxy` binary in the same folder as your `manifest.yml`. Also ensure the `logproxy` binary has *executable* privileges. (you can use the `chmod a+x logproxy` command on Linux based shells to achieve the result) 


```
applications:
- name: logproxy
  domain: your-domain.com
  instances: 2
  memory: 128M
  disk_quota: 128M
  routes:
  - route: logproxy.your-domain.com
  buildpack: binary_buildpack
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

A `Procfile` is required as well:

```
web: logproxy
```

Now push the application:

```
cf push
```

If everything went OK logproxy should now be reachable on https://logproxy.your-domain.com . The logdrain endpoint would then be:

```
https://logproxy.your-domain.com/syslog/drain/RandomTokenHere
```

# Configure logdrains
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

# TODO
- Better handling of HTTP 635 errors
- Retry mechanism in case of POST failures 
