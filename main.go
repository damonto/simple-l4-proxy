package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type proxies []string

func (p *proxies) String() string {
	return ""
}

func (p *proxies) Set(value string) error {
	*p = append(*p, value)
	return nil
}

var proxyConf proxies

func init() {
	flag.Var(&proxyConf, "p", "-p 127.0.0.1:80:10.0.0.1:8081")

	flag.Parse()
}

type forwarder struct {
	Local  string
	Remote string
}

func (f *forwarder) Forward() {
	ln, err := net.Listen("tcp", f.Local)
	if err != nil {
		logrus.Errorf("Failed to listen local address %s", f.Local)
		os.Exit(1)
	}
	fmt.Printf("Proxy server running on %s \n", f.Local)

	for {
		if conn, err := ln.Accept(); err == nil {
			rconn, err := net.DialTimeout("tcp", f.Remote, time.Second*1)
			if err != nil {
				logrus.Errorf("Establish connection with remote server %s: %v", f.Remote, err)
				conn.Close() // close manual
				continue
			}

			logrus.Infof("Established connection %s \n", conn.RemoteAddr())

			f.Copy(conn, rconn)
		}
	}
}

func (f *forwarder) Copy(lconn, rconn net.Conn) {
	defer rconn.Close()
	defer lconn.Close()

	go func() {
		n, err := io.Copy(lconn, rconn)
		logrus.Infof("Copied %d bytes from %s to %s %v \n", n, f.Local, f.Remote, err)
	}()
	n, err := io.Copy(rconn, lconn)
	logrus.Infof("Copied %d bytes from %s to %s %v \n", n, f.Remote, f.Local, err)
}

func main() {
	for _, proxy := range proxyConf {
		conf := strings.Split(proxy, ":")

		f := &forwarder{}
		if len(conf) == 3 {
			f = &forwarder{
				Local:  ":" + conf[0],
				Remote: conf[1] + ":" + conf[2],
			}
		} else {
			f = &forwarder{
				Local:  conf[0] + ":" + conf[1],
				Remote: conf[2] + ":" + conf[3],
			}
		}

		go f.Forward()
	}

	select {}
}
