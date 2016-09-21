package gbench

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	rpc "github.com/relab/gorums/gridq"
	"github.com/tylertreat/bench"

	"google.golang.org/grpc"
)

// GorumsRequesterFactory implements RequesterFactory by creating a Requester which
// issues requests to a register using the Gorums framework.
type GridQRequesterFactory struct {
	Addrs             []string
	ReadQuorum        int
	WriteQuorum       int
	PayloadSize       int
	QRPCTimeout       time.Duration
	WriteRatioPercent int
}

// GetRequester returns a new Requester, called for each Benchmark connection.
func (r *GridQRequesterFactory) GetRequester(uint64) bench.Requester {
	return &gridqRequester{
		addrs:       r.Addrs,
		readq:       r.ReadQuorum,
		writeq:      r.WriteQuorum,
		payloadSize: r.PayloadSize,
		timeout:     r.QRPCTimeout,
		writeRatio:  r.WriteRatioPercent,
		dialOpts: []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithBlock(),
			grpc.WithTimeout(time.Second),
		},
	}
}

type gridqRequester struct {
	addrs       []string
	readq       int
	writeq      int
	payloadSize int
	timeout     time.Duration
	writeRatio  int

	dialOpts []grpc.DialOption

	mgr    *rpc.Manager
	config *rpc.Configuration
	state  *rpc.State
}

func (gr *gridqRequester) Setup() error {
	var err error
	gr.mgr, err = rpc.NewManager(
		gr.addrs,
		rpc.WithGrpcDialOptions(gr.dialOpts...),
	)
	if err != nil {
		return err
	}

	ids := gr.mgr.NodeIDs()
	qspec := rpc.NewGQSortX(gr.readq, gr.writeq)
	gr.config, err = gr.mgr.NewConfiguration(ids, qspec, gr.timeout)
	if err != nil {
		return err
	}

	// Set initial state.
	gr.state = &rpc.State{
		Value:     strings.Repeat("x", gr.payloadSize),
		Timestamp: time.Now().UnixNano(),
	}
	wreply, err := gr.config.Write(gr.state)
	if err != nil {
		return fmt.Errorf("write rpc error: %v", err)
	}
	if !wreply.Reply.New {
		return fmt.Errorf("intital write reply was not marked as new")
	}

	return nil
}

func (gr *gridqRequester) Request() error {
	var err error
	switch gr.writeRatio {
	case 0:
		_, err = gr.config.Read(&rpc.Empty{})
	case 100:
		gr.state.Timestamp = time.Now().UnixNano()
		_, err = gr.config.Write(gr.state)
	default:
		x := rand.Intn(100)
		if x < gr.writeRatio {
			gr.state.Timestamp = time.Now().UnixNano()
			_, err = gr.config.Write(gr.state)
		} else {
			_, err = gr.config.Read(&rpc.Empty{})
		}
	}

	return err
}

func (gr *gridqRequester) Teardown() error {
	gr.mgr.Close()
	gr.mgr = nil
	gr.config = nil
	return nil
}
