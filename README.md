# Avenues
[![GitHub Workflow Status](https://img.shields.io/github/workflow/status/gomicro/avenues/Build/master)](https://github.com/gomicro/avenues/actions?query=workflow%3ABuild+branch%3Amaster)
[![Go Reportcard](https://goreportcard.com/badge/github.com/gomicro/avenues)](https://goreportcard.com/report/github.com/gomicro/avenues)
[![GoDoc](https://godoc.org/github.com/gomicro/avenues?status.svg)](https://godoc.org/github.com/gomicro/avenues)
[![License](https://img.shields.io/github/license/gomicro/avenues.svg)](https://github.com/gomicro/avenues/blob/master/LICENSE.md)
[![Release](https://img.shields.io/github/release/gomicro/avenues.svg)](https://github.com/gomicro/avenues/releases/latest)

Avenues is specifically for creating a singular API layer out of several dockerized Microservices.  More specifically for local testing.  It is in no way expected to take a production load.  Ideally in production you'll have a load balancer or some other service that has been significantly battle tested to demonstrate reliability. Beyond that, you typically don't have a load balancer or these other things when testing locally, and it is very nice to test your entire API holistically rather than as individual services. Avenues allows for you to provide a config for the routing, and will proxy all requests to the designated services. The benefit coming from it's simplicity and being light weight.  The container houses only the app, and as such is a very tiny docker image.

# Requirements
Docker

# Usage

## Configuration
The configuration always looks to read from a `routes.yaml` file.  It expects one segment, definition of given routes.

The endpoints are treated as the root of the endpoint, so all sub paths of the routes specified will direct to those routes as well.  i.e. `/v1/teams` will match for `/v1/teams`, `/v1/teams/{teamID}`, `/v1/teams/{teamID}/admin`, and so on.

```
routes:
  "/v1/projects":
    type: "static"
    backend: "http://service1:4567"
  "/v1/users":
    backend: "http://service2:4567"
  "/v1/teams":
    backend: "http://service2:4567"
  "/v1/posts":
    type: "ordinal"
    backends:
      - "http://service3:4567"
      - "http://anothermockofservice3:4567"
      - "http://mockfailureservice:4567"
reset: "/a/custom/path/for/reset" # Optional
status: "/a/custom/path/for/status" # Optional
cert: "cert for serving ssl" # Optional
cert_path: "path to file containing cert" # Optional
key: "key for serving ssl" # Optional
key_path: "path to file containing key" # Optional
ca: "a custom CA to include for SSL" # Optional
ca_path: "path to file containing CA(.)(.)" # Optional
```

## Running
Avenues is intended to be used in conjunction with local Docker testing of a service.

```
docker pull ghcr.io/gomicro/avenues
docker run -it -v $PWD/routes.yaml:/routes.yaml ghcr.io/gomicro/avenues
```

# Versioning
The app will be versioned in accordance with [Semver 2.0.0](http://semver.org).  See the [releases](https://github.com/gomicro/avenues/releases) section for the latest version.  Until version 1.0.0 the app is considered to be unstable.

# License
See [LICENSE.md](./LICENSE.md) for more information.
