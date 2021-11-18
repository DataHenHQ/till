package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/DataHenHQ/till/internal/tillclient"
)

func newInstanceStatMutex() tillclient.InstanceStatMutex {
	return tillclient.InstanceStatMutex{
		Mutex: &sync.Mutex{},
		InstanceStat: tillclient.InstanceStat{
			Requests:            newZeroStat(),
			SuccessfulRequests:  newZeroStat(),
			FailedRequests:      newZeroStat(),
			InterceptedRequests: newZeroStat(),
			CacheHits:           newZeroStat(),
			CacheSets:           newZeroStat(),
			Name:                &InstanceName,
		},
	}
}

func startRecurringStatUpdate() {
	client, err := tillclient.NewClient(Token)
	if err != nil {
		log.Fatal(err)
	}

	for {
		time.Sleep(time.Second * 5)
		// Take a snapshot of the state of the instate stat by doing deep copy
		is := StatMu.InstanceStat.DeepCopy()

		// if instance stat is zero then skip this step
		if is.IsZero() {
			continue
		}

		// Update the stat on the cloud
		i, _, err := client.InstanceStats.Update(context.Background(), is)
		if err != nil {
			fmt.Printf("gotten error: %v\n", err)
			continue
		}

		// set the current instance global var
		curri = *i

		resetInstanceStatDelta(is)

	}
}
