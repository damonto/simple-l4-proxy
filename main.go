package main

import (
	"flag"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
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
	local  string
	remote string
}

func (f *forwarder) Forward() {
	ln, err := net.Listen("tcp", f.local)
	if err != nil {
		logrus.Errorf("failed to listen local address %s", f.local)
		panic(err)
	}
	logrus.Infof("proxy server running on %s \n", f.local)

	for {
		if conn, err := ln.Accept(); err == nil {
			go func(conn net.Conn) {
				err := f.Handle(conn)
				if err != nil {
					logrus.Errorf("failed to establish connection %v", err)
				}
			}(conn)
		}
	}
}

func (f *forwarder) Handle(conn net.Conn) error {
	defer conn.Close()
	dst, err := net.DialTimeout("tcp", f.remote, time.Second)
	if err != nil {
		return err
	}
	defer dst.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go f.Copy(dst, conn, &wg)
	go f.Copy(conn, dst, &wg)

	wg.Wait()

	return nil
}

func (f *forwarder) Copy(src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()

	n, err := io.Copy(dst, src)
	logrus.Infof("%s src->dst %s %d bytes %v", src.RemoteAddr().String(), dst.RemoteAddr().String(), n, err)
}

func main() {
	// CPU Cores * 2
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	for _, proxy := range proxyConf {
		conf := strings.Split(proxy, ":")

		f := &forwarder{}
		if len(conf) == 3 {
			f = &forwarder{
				local:  ":" + conf[0],
				remote: conf[1] + ":" + conf[2],
			}
		} else {
			f = &forwarder{
				local:  conf[0] + ":" + conf[1],
				remote: conf[2] + ":" + conf[3],
			}
		}

		go f.Forward()
	}

	select {}
}
