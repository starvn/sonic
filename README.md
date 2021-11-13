<img src="./docs/image/sonic-api-gateway.png" alt="Sonic API Gateway" width="20%"/>

# Sonic API Gateway
![Build](https://github.com/starvn/sonic/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/sonic)](https://goreportcard.com/report/github.com/starvn/sonic) [![GoDoc](https://godoc.org/github.com/starvn/sonic?status.svg)](https://godoc.org/github.com/starvn/sonic)

Sonic is an API Gateway that aggregates and manipulates multiple data sources coming from your microservices to provide the exact API your end-user products need while offering awesome performance.

## Installation
Sonic is a single binary file that does not require any external libraries to work. To install Sonic choose your operative system in the downloads section or use the Docker image.

Generate your sonic.json here or use the [sample file](sonic.json) (updating...)

### Docker

If you are already familiar with Docker, the easiest way to get started is by pulling and running the Sonic image from the Docker Hub.

Share your configuration as a volume (mapped to /etc/sonic). Inside the volume you need at least the sonic.json file which contains the endpoint definition of you application.

**Pull the image and run Sonic (default parameters):**

```
docker pull starvn/sonic:tagname
docker run -p 8080:8080 -v $PWD:/etc/sonic/ starvn/sonic:tagname
```

**Run with the debug enabled (flag -d):**

```
docker run -p 8080:8080 -v "${PWD}:/etc/sonic/" starvn/sonic:tagname run -d -c /etc/sonic/sonic.json
```

**When the container is running with the previous line you can access the health endpoint:**

```
curl HTTP://localhost:8080/__health
```

**Check the syntax of the configuration file:**

```
docker run -it -p 8080:8080 -v $PWD:/etc/sonic/ starvn/sonic:tagname check --config sonic.json
```

**Show the help:**

```
docker run -it -p 8080:8080 -v $PWD:/etc/sonic/ starvn/sonic:tagname --help
```


## Development setup
You need Golang >= 1.17

**Note**: if you are using windows, we recommend you to build the project on linux

### Fork the sonic project
Go to the [sonic](https://github.com/starvn/sonic) project and click on the "fork" button. You can then clone your own fork of the project, and start working on it.

[Please read the Github forking documentation for more information](https://help.github.com/articles/fork-a-repo).

### Build

```
make build
```

### Build with docker

```
make build_on_docker
```
