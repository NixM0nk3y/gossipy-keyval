package main

import (
	"context"
	"flag"
	"fmt"
	"gossipy/internal/api"
	"gossipy/internal/cluster"
	"gossipy/internal/store"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func wait_signal(cancel context.CancelFunc) {
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		select {
		case s := <-signal_chan:
			log.Printf("signal %s happen", s.String())
			cancel()
		}
	}
}

func main() {

	//
	var hostname string = "node1"
	val, ok := os.LookupEnv("HOSTNAME")
	if ok {
		hostname = val
	}

	// setup our flags
	debug := flag.Bool("debug", false, "sets log level to debug")
	apiPort := flag.Int("apiport", 7080, "default api port to listen on")
	retrySecs := flag.Int64("retrysecs", 300, "maximum number of secs to wait to join cluster")
	clusterIP := flag.String("clusterip", "127.0.0.1", "default cluster ip to communicate on")
	clusterPort := flag.Int("clusterport", 7947, "default api port to listen on")
	clusterNode := flag.String("clusternode", hostname, "default node name")
	serviceDiscoveryHost := flag.String("servicediscoveryhost", "localhost", "default node name")

	flag.Parse()

	// initialise our logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Info().Msg("starting gossipy")

	// initialise our key store
	kvstore := store.NewKeyValueStore()

	// initialise our node, message channel and cluster
	msgCh := make(chan []byte)

	node := &cluster.Node{Name: *clusterNode, Addr: *clusterIP, Port: *clusterPort}
	kvcluster, err := cluster.NewCluster(node, kvstore, msgCh)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed to create cluster")
	}

	// setup our backoff algorithm
	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = 1 * time.Second
	backOff.Multiplier = 2.0
	backOff.MaxInterval = 10 * time.Second
	backOff.MaxElapsedTime = time.Duration(*retrySecs) * time.Second
	backOff.Reset()

	// setup our connection function
	operation := func() error {
		// lookup our cluster candidates via a SRV record
		_, addrs, err := net.LookupSRV("", "", *serviceDiscoveryHost)
		if err != nil {
			return err
		}

		// build our list of potential cluster candidates
		var clusterSeeds []string
		for _, host := range addrs {
			log.Info().Msgf("discovery host: %s", host.Target)
			clusterSeeds = append(clusterSeeds, host.Target)
		}

		// attempt to join the cluster
		return kvcluster.Join(clusterSeeds)
	}

	// feedback for progress
	notify := func(err error, time time.Duration) {
		log.Error().Msgf("Leadership election error %+v, retrying in %s", err, time)
	}

	// run our connection loop
	err = backoff.RetryNotify(operation, backOff, notify)

	// our final status
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed to join cluster")
	}

	// record our cluster members
	for _, member := range kvcluster.Members() {
		log.Info().Msgf("Cluster Member: %s %s\n", member.Name, member.Addr)
	}

	// finally start our API
	api := api.NewAPI(kvcluster, kvstore)

	// run our API in a separate thread
	apiDone := make(chan bool)
	go func() {
		err := api.Run(fmt.Sprintf(":%d", *apiPort))
		if err != nil {
			log.Error().Stack().Err(err).Msg("failed to start API")
		}
		apiDone <- true
	}()

	// setup our signal handling and cancellation context
	stopCtx, cancel := context.WithCancel(context.TODO())
	go wait_signal(cancel)

	tick := time.NewTicker(30 * time.Second)
	run := true
	for run {
		select {
		// show our cluster members
		case <-tick.C:
			for _, member := range kvcluster.Members() {
				log.Info().Msgf("Cluster Member: %s %s\n", member.Name, member.Addr)
			}
		// received a broadcast message
		case data := <-msgCh:
			msg, ok := cluster.ParseMyBroadcastMessage(data)
			if !ok {
				continue
			}

			log.Info().Msgf("received broadcast msg: operation=%s key=%s value=%s", msg.Operation, msg.Key, msg.Value)

			switch msg.Operation {
			case "delete":
				kvstore.Delete(msg.Key)
			case "set":
				kvstore.Set(msg.Key, msg.Value)
			default:
				log.Warn().Msgf("unknown operation=%s received", msg.Operation)
			}

		// API died
		case <-apiDone:
			log.Warn().Msg("API stopped")
			run = false
		// received a stop signal
		case <-stopCtx.Done():
			log.Warn().Msg("stop signal called")
			run = false
		}
	}

	tick.Stop()

	// exit the cluster
	err = kvcluster.Leave(time.Second * 5)
	if err != nil {
		log.Error().Stack().Err(err).Msg("failed leaving the cluster")
	}

	log.Info().Msg("shutting down")
	os.Exit(0)

}
