package libp2p

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/libp2p/go-libp2p-core/network"
	rcmgr "github.com/libp2p/go-libp2p-resource-manager"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingResourceManager(t *testing.T) {
	clock := clock.NewMock()
	limiter := rcmgr.NewDefaultLimiter()
	limiter.SystemLimits = limiter.SystemLimits.WithConnLimit(1, 1, 1)
	rm, err := rcmgr.NewResourceManager(limiter)
	if err != nil {
		t.Fatal(err)
	}

	oCore, oLogs := observer.New(zap.WarnLevel)
	oLogger := zap.New(oCore)
	lrm := &loggingResourceManager{
		clock:       clock,
		logger:      oLogger.Sugar(),
		delegate:    rm,
		logInterval: 1 * time.Second,
	}

	// 2 of these should result in resource limit exceeded errors and subsequent log messages
	for i := 0; i < 3; i++ {
		_, _ = lrm.OpenConnection(network.DirInbound, false)
	}

	// run the logger which will write an entry for those errors
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lrm.start(ctx)
	clock.Add(3 * time.Second)

	timer := time.NewTimer(1 * time.Second)
	for {
		select {
		case <-timer.C:
			t.Fatalf("expected logs never arrived")
		default:
			if oLogs.Len() == 0 {
				continue
			}
			require.Equal(t, "Resource limits were exceeded 2 times, consider inspecting logs and raising the resource manager limits.", oLogs.All()[0].Message)
			return
		}
	}
}
