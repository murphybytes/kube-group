package group

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"
)

const DefaultCheckFrequecy = time.Second

func LocalIP() (string, error) {
	// attempt to establish outbound connection to Google DNS
	// so we can resolve the IP address used to make the connection
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("attempt to connect to derive local ip failed %w", err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", errors.New("internal error could not cast local address to expected type")
	}

	return localAddr.IP.String(), nil
}

func GroupIPs(serviceName string) ([]string, error) {
	ips, err := net.LookupIP(serviceName)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve peer ip addresses. %w", err)
	}

	var results []string
	for _, ip := range ips {
		results = append(results, ip.String())
	}
	return results, nil
}

type NotifyFn func(ips []string) error

type optionCfg struct {
	checkFrequency time.Duration
}

type WatchOptionFn func(*optionCfg)

func WithCheckFrequency(t time.Duration) func(*optionCfg) {
	return func(cfg *optionCfg) {
		cfg.checkFrequency = t
	}
}

func Watch(ctx context.Context, service string, notifier NotifyFn, options ...WatchOptionFn) (context.CancelFunc, error) {
	cfg := optionCfg{
		checkFrequency: DefaultCheckFrequecy,
	}
	for _, optFn := range options {
		optFn(&cfg)
	}

	// test to see if the dns name for the service exists
	if _, err := GroupIPs(service); err != nil {
		return nil, fmt.Errorf("could not get IPs for service %q %w", service, err)
	}

	newCtx, cancelFn := context.WithCancel(ctx)
	groupMemberFn := func(ctx context.Context) ([]string, error) {
		return GroupIPs(service)
	}
	ticker := time.NewTicker(cfg.checkFrequency)

	go watch(newCtx, notifier, groupMemberFn, ticker.C)

	return cancelFn, nil
}

func watch(ctx context.Context, notifyFn NotifyFn,
	groupFn func(context.Context) ([]string, error), c <-chan time.Time) {

	var previousAddrs string

	for {
		select {
		case <-ctx.Done():
			return
		case <-c:
			currentAddrs, err := groupFn(ctx)
			if err != nil {
				log.Fatalf("could not fetch ips of pods %s", err)
			}
			// create a string from ip addresses that we
			// can use to compare this set of IPs with the
			// last set we got so we can tell if anything has
			// changed
			addrs := currentAddrs
			sort.Strings(addrs)
			s := strings.Join(addrs, "")

			if s != previousAddrs {
				previousAddrs = s

				if err := notifyFn(currentAddrs); err != nil {
					log.Printf("notify function failed %s", err)
				}
			}

		}
	}
}
