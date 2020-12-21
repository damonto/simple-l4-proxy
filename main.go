package main

import (
	"flag"
	"io"
	"net"
	"runtime"
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
		panic(err)
	}
	logrus.Infof("Proxy server running on %s \n", f.Local)

	for {
		if conn, err := ln.Accept(); err == nil {
			dst, err := net.DialTimeout("tcp", f.Remote, time.Millisecond*500)
			if err != nil {
				logrus.Errorf("Establish connection with remote server %s: %v", f.Remote, err)
				conn.Close() // close manual
				continue
			}

			logrus.Infof("Established connection %s %s src->dst %s\n", conn.RemoteAddr(), f.Local, f.Remote)

			f.Copy(conn, dst)
		}
	}
}

func (f *forwarder) Copy(src, dst net.Conn) {
	defer dst.Close()
	defer src.Close()

	done := make(chan struct{})

	go func() {
		n, err := io.Copy(dst, src)
		logrus.Infof("Copied %d bytes from %s to %s %v \n", n, f.Local, f.Remote, err)
		done <- struct{}{}
	}()

	n, err := io.Copy(src, dst)
	logrus.Infof("Copied %d bytes from %s to %s %v \n", n, f.Remote, f.Local, err)

	<-done
}

func main() {
	// CPU Cores * 2
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

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
