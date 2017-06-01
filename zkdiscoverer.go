package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
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
	oneshot := flag.Bool("oneshot", false, "register for alltime and exit")
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

	if *oneshot {
		c, _, err := zk.Connect(strings.Split(*zookeepers, ","), time.Millisecond*time.Duration(4000))
		if err != nil {
			panic(err)
		}
		acl := zk.WorldACL(zk.PermAll)

		_ = c
		_ = acl

		createIfNotExists := func(path string) *zk.Stat {
			exists, stat, err := c.Exists(path)
			if err != nil {
				fmt.Printf("zknode %s check error\n", path)
				panic(err)
			}
			if exists {
				return stat
			}
			_, err = c.Create(path, []byte{}, 0, acl)
			if err != nil {
				fmt.Printf("zknode %s create error\n", path)
				panic(err)
			}
			return nil
		}

		prefix := "/discovery"
		createIfNotExists(prefix)
		createIfNotExists(fmt.Sprintf("%s/%s", prefix, env))
		createIfNotExists(fmt.Sprintf("%s/%s/%s", prefix, env, *service))

		hostNodePath := fmt.Sprintf("%s/%s/%s/%s", prefix, env, *service, hostname)
		createIfNotExists(hostNodePath)

		data, stat, err := c.Get(hostNodePath)
		if err != nil {
			fmt.Printf("zknode %s get error\n", hostNodePath)
			panic(err)
		}
		ver := stat.Version

		data2 := []byte(fmt.Sprintf(`{"serviceEndpoint":{"host":"%s","port":%d},"additionalEndpoints":{},"status":"ALIVE"}`, *host, *port))

		if bytes.Equal(data, data2) {
			fmt.Println("Service already registered")
		} else {
			_, err = c.Set(hostNodePath, data2, ver)
			if err != nil {
				fmt.Printf("zknode %s set error\n", hostNodePath)
				panic(err)
			}
			fmt.Printf("Service %s registered: %s:%d\n", *service, *host, *port)
		}

		os.Exit(0)
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
