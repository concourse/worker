package sweeper

import (
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/concourse/atc"
	"github.com/concourse/worker/beacon"
	"github.com/tedsuo/ifrit"
)

// Command is the struct that holds the properties for the mark and sweep command
type Command struct {
	Logger       lager.Logger
	BeaconClient beacon.BeaconClient
	GCInterval   time.Duration
}

// NewSweeperRunner provides the ifrit runner that marks and sweeps the containers
func NewSweeperRunner(logger lager.Logger, atcWorker atc.Worker, config beacon.Config) ifrit.Runner {
	logger.Info("sweep-starting")

	client := beacon.NewSSHClient(logger.Session("beacon-client"), config)

	beaconC := &beacon.Beacon{
		Logger:    logger,
		Worker:    atcWorker,
		Client:    client,
		KeepAlive: false, // disable keepalive for mark and sweep calls
	}

	scmd := &Command{
		BeaconClient: beaconC,
		Logger:       logger.Session("sweeper"),
		GCInterval:   30 * time.Second,
	}
	return scmd
}

// Run invokes the process of marking and sweeping containers
// First worker will call atc to collect list of containers to be removed
// and then worker will report its state of current containers for
// atc to remove containers in DB. This cycle is triggered every GCInterval sec
func (cmd *Command) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	timer := time.NewTicker(cmd.GCInterval)
	close(ready)

	for {
		select {
		case <-timer.C:
			err := cmd.BeaconClient.MarkandSweepContainers()
			if err != nil {
				cmd.Logger.Error("failed-to-mark-and-swep-containers", err)
			}
		case <-signals:
			cmd.Logger.Info("exiting-from-mark-and-sweep")
			return nil
		}
	}
}
