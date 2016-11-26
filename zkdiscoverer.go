package main

import (
	"flag"
	"fmt"
	"github.com/strava/go.serversets"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	hostname, _ := os.Hostname()

	zookeepers := flag.String("zookeepers", "localhost:2181", "zookeeper endpoints joined by ','")
	service := flag.String("service", "", "service name")
	environment := flag.String("environment", "local", "service environment (local, staging, test or production")
	host := flag.String("host", hostname, "service host")
	port := flag.Int("port", -1, "service port")
	flag.Parse()

	env := serversets.Local

	switch *environment {
	case "local":
		env = serversets.Local
	case "staging":
		env = serversets.Staging
	case "test":
		env = serversets.Test
	case "production":
		env = serversets.Production
	default:
		panic("Wrong environment")
	}

	if *service == "" {
		panic("Service not specified")
	}

	serverSet := serversets.New(env, *service, strings.Split(*zookeepers, ","))

	pingFunction := func() error {
		_, err := net.DialTimeout("tcp", net.JoinHostPort(*host, strconv.Itoa(*port)), 1*time.Second)
		if err != nil {
			fmt.Println(err)
		}
		return err
	}

	endpoint, err := serverSet.RegisterEndpoint(*host, *port, pingFunction)
	fmt.Println(endpoint)
	if err != nil {
		fmt.Println(err)
	}

	for {
		time.Sleep(60 * time.Second)
	}
}
