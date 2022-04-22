package pkg

import (
	"context"
	"net/http/httptrace"

	"github.com/sirupsen/logrus"
)

func TracedContext(ctx context.Context) context.Context {
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			logrus.Debugf("Got Conn: %+v\n", connInfo)
		},
		ConnectStart: func(network, addr string) {
			logrus.Debugf("Conn Started: %s %s\n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				logrus.Errorf("Conn Error: %s %s %s\n", network, addr, err)
			} else {
				logrus.Debugf("Conn Done: %s %s\n", network, addr)
			}
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			logrus.Debugf("DNS Info: %+v\n", dnsInfo)
		},
	}
	return httptrace.WithClientTrace(ctx, trace)
}
