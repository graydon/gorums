// Code generated by 'gorums' plugin for protoc-gen-go. DO NOT EDIT.
// Source file to edit is: calltype_future_tmpl

package dev

import (
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* Exported types and methods for asynchronous quorum call method ReadFuture */

// ReadFutureReply is a future object for an asynchronous ReadFuture quorum call invocation.
type ReadFutureReply struct {
	// the actual reply
	*State
	NodeIDs []uint32
	err     error
	c       chan struct{}
}

// ReadFuture asynchronously invokes a quorum call on configuration c
// and returns a ReadFutureReply which can be used to inspect the quorum call
// reply and error when available.
func (c *Configuration) ReadFuture(ctx context.Context, arg *ReadRequest) *ReadFutureReply {
	f := &ReadFutureReply{
		NodeIDs: make([]uint32, 0, c.n),
		c:       make(chan struct{}, 1),
	}
	go func() {
		defer close(f.c)
		c.readFuture(ctx, arg, f)
	}()
	return f
}

// Get returns the reply and any error associated with the ReadFuture.
// The method blocks until a reply or error is available.
func (f *ReadFutureReply) Get() (*State, error) {
	<-f.c
	return f.State, f.err
}

// Done reports if a reply and/or error is available for the ReadFuture.
func (f *ReadFutureReply) Done() bool {
	select {
	case <-f.c:
		return true
	default:
		return false
	}
}

/* Unexported types and methods for asynchronous quorum call method ReadFuture */

type readFutureReply struct {
	nid   uint32
	reply *State
	err   error
}

func (c *Configuration) readFuture(ctx context.Context, a *ReadRequest, resp *ReadFutureReply) {
	var ti traceInfo
	if c.mgr.opts.trace {
		ti.tr = trace.New("gorums."+c.tstring()+".Sent", "ReadFuture")
		defer ti.tr.Finish()

		ti.firstLine.cid = c.id
		if deadline, ok := ctx.Deadline(); ok {
			ti.firstLine.deadline = deadline.Sub(time.Now())
		}
		ti.tr.LazyLog(&ti.firstLine, false)
		ti.tr.LazyLog(&payload{sent: true, msg: a}, false)

		defer func() {
			ti.tr.LazyLog(&qcresult{
				ids:   resp.NodeIDs,
				reply: resp.State,
				err:   resp.err,
			}, false)
			if resp.err != nil {
				ti.tr.SetError()
			}
		}()
	}

	replyChan := make(chan readFutureReply, c.n)
	for _, n := range c.nodes {
		go callGRPCReadFuture(ctx, n, a, replyChan)
	}

	var (
		replyValues = make([]*State, 0, c.n)
		reply       *State
		errCount    int
		quorum      bool
	)

	for {
		select {
		case r := <-replyChan:
			resp.NodeIDs = append(resp.NodeIDs, r.nid)
			if r.err != nil {
				errCount++
				break
			}
			if c.mgr.opts.trace {
				ti.tr.LazyLog(&payload{sent: false, id: r.nid, msg: r.reply}, false)
			}
			replyValues = append(replyValues, r.reply)
			if reply, quorum = c.qspec.ReadFutureQF(replyValues); quorum {
				resp.State, resp.err = reply, nil
				return
			}
		case <-ctx.Done():
			resp.State, resp.err = reply, QuorumCallError{ctx.Err().Error(), errCount, len(replyValues)}
			return
		}

		if errCount+len(replyValues) == c.n {
			resp.State, resp.err = reply, QuorumCallError{"incomplete call", errCount, len(replyValues)}
			return
		}
	}
}

func callGRPCReadFuture(ctx context.Context, node *Node, arg *ReadRequest, replyChan chan<- readFutureReply) {
	if arg == nil {
		// send a nil reply to the for-select-loop
		replyChan <- readFutureReply{node.id, nil, nil}
		return
	}
	reply := new(State)
	start := time.Now()
	err := grpc.Invoke(
		ctx,
		"/dev.Storage/ReadFuture",
		arg,
		reply,
		node.conn,
	)
	s, ok := status.FromError(err)
	if ok && (s.Code() == codes.OK || s.Code() == codes.Canceled) {
		node.setLatency(time.Since(start))
	} else {
		node.setLastErr(err)
	}
	replyChan <- readFutureReply{node.id, reply, err}
}

/* Exported types and methods for asynchronous quorum call method WriteFuture */

// WriteFutureReply is a future object for an asynchronous WriteFuture quorum call invocation.
type WriteFutureReply struct {
	// the actual reply
	*WriteResponse
	NodeIDs []uint32
	err     error
	c       chan struct{}
}

// WriteFuture asynchronously invokes a quorum call on configuration c
// and returns a WriteFutureReply which can be used to inspect the quorum call
// reply and error when available.
func (c *Configuration) WriteFuture(ctx context.Context, arg *State) *WriteFutureReply {
	f := &WriteFutureReply{
		NodeIDs: make([]uint32, 0, c.n),
		c:       make(chan struct{}, 1),
	}
	go func() {
		defer close(f.c)
		c.writeFuture(ctx, arg, f)
	}()
	return f
}

// Get returns the reply and any error associated with the WriteFuture.
// The method blocks until a reply or error is available.
func (f *WriteFutureReply) Get() (*WriteResponse, error) {
	<-f.c
	return f.WriteResponse, f.err
}

// Done reports if a reply and/or error is available for the WriteFuture.
func (f *WriteFutureReply) Done() bool {
	select {
	case <-f.c:
		return true
	default:
		return false
	}
}

/* Unexported types and methods for asynchronous quorum call method WriteFuture */

type writeFutureReply struct {
	nid   uint32
	reply *WriteResponse
	err   error
}

func (c *Configuration) writeFuture(ctx context.Context, a *State, resp *WriteFutureReply) {
	var ti traceInfo
	if c.mgr.opts.trace {
		ti.tr = trace.New("gorums."+c.tstring()+".Sent", "WriteFuture")
		defer ti.tr.Finish()

		ti.firstLine.cid = c.id
		if deadline, ok := ctx.Deadline(); ok {
			ti.firstLine.deadline = deadline.Sub(time.Now())
		}
		ti.tr.LazyLog(&ti.firstLine, false)
		ti.tr.LazyLog(&payload{sent: true, msg: a}, false)

		defer func() {
			ti.tr.LazyLog(&qcresult{
				ids:   resp.NodeIDs,
				reply: resp.WriteResponse,
				err:   resp.err,
			}, false)
			if resp.err != nil {
				ti.tr.SetError()
			}
		}()
	}

	replyChan := make(chan writeFutureReply, c.n)
	for _, n := range c.nodes {
		go callGRPCWriteFuture(ctx, n, a, replyChan)
	}

	var (
		replyValues = make([]*WriteResponse, 0, c.n)
		reply       *WriteResponse
		errCount    int
		quorum      bool
	)

	for {
		select {
		case r := <-replyChan:
			resp.NodeIDs = append(resp.NodeIDs, r.nid)
			if r.err != nil {
				errCount++
				break
			}
			if c.mgr.opts.trace {
				ti.tr.LazyLog(&payload{sent: false, id: r.nid, msg: r.reply}, false)
			}
			replyValues = append(replyValues, r.reply)
			if reply, quorum = c.qspec.WriteFutureQF(a, replyValues); quorum {
				resp.WriteResponse, resp.err = reply, nil
				return
			}
		case <-ctx.Done():
			resp.WriteResponse, resp.err = reply, QuorumCallError{ctx.Err().Error(), errCount, len(replyValues)}
			return
		}

		if errCount+len(replyValues) == c.n {
			resp.WriteResponse, resp.err = reply, QuorumCallError{"incomplete call", errCount, len(replyValues)}
			return
		}
	}
}

func callGRPCWriteFuture(ctx context.Context, node *Node, arg *State, replyChan chan<- writeFutureReply) {
	if arg == nil {
		// send a nil reply to the for-select-loop
		replyChan <- writeFutureReply{node.id, nil, nil}
		return
	}
	reply := new(WriteResponse)
	start := time.Now()
	err := grpc.Invoke(
		ctx,
		"/dev.Storage/WriteFuture",
		arg,
		reply,
		node.conn,
	)
	s, ok := status.FromError(err)
	if ok && (s.Code() == codes.OK || s.Code() == codes.Canceled) {
		node.setLatency(time.Since(start))
	} else {
		node.setLastErr(err)
	}
	replyChan <- writeFutureReply{node.id, reply, err}
}
