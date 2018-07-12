// Code generated by protoc-gen-gorums. DO NOT EDIT.
// Source file to edit is: dev/storage.proto
// Template file to edit is: calltype_future.tmpl

package dev

import (
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* Exported asynchronous quorum call method ReadFuture */

// ReadFuture asynchronously invokes a quorum call on configuration c
// and returns a FutureState which can be used to inspect the quorum call
// reply and error when available.
func (c *Configuration) ReadFuture(ctx context.Context, arg *ReadRequest) *FutureState {
	f := &FutureState{
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
func (f *FutureState) Get() (*State, error) {
	<-f.c
	return f.State, f.err
}

// Done reports if a reply and/or error is available for the ReadFuture.
func (f *FutureState) Done() bool {
	select {
	case <-f.c:
		return true
	default:
		return false
	}
}

/* Unexported asynchronous quorum call method ReadFuture */

func (c *Configuration) readFuture(ctx context.Context, a *ReadRequest, resp *FutureState) {
	var ti traceInfo
	if c.mgr.opts.trace {
		ti.Trace = trace.New("gorums."+c.tstring()+".Sent", "ReadFuture")
		defer ti.Finish()

		ti.firstLine.cid = c.id
		if deadline, ok := ctx.Deadline(); ok {
			ti.firstLine.deadline = time.Until(deadline)
		}
		ti.LazyLog(&ti.firstLine, false)
		ti.LazyLog(&payload{sent: true, msg: a}, false)

		defer func() {
			ti.LazyLog(&qcresult{
				ids:   resp.NodeIDs,
				reply: resp.State,
				err:   resp.err,
			}, false)
			if resp.err != nil {
				ti.SetError()
			}
		}()
	}

	expected := c.n
	replyChan := make(chan internalState, expected)
	for _, n := range c.nodes {
		go callGRPCReadFuture(ctx, n, a, replyChan)
	}

	var (
		replyValues = make([]*State, 0, c.n)
		reply       *State
		errs        []GRPCError
		quorum      bool
	)

	for {
		select {
		case r := <-replyChan:
			resp.NodeIDs = append(resp.NodeIDs, r.nid)
			if r.err != nil {
				errs = append(errs, GRPCError{r.nid, r.err})
				break
			}
			if c.mgr.opts.trace {
				ti.LazyLog(&payload{sent: false, id: r.nid, msg: r.reply}, false)
			}
			replyValues = append(replyValues, r.reply)
			if reply, quorum = c.qspec.ReadFutureQF(replyValues); quorum {
				resp.State, resp.err = reply, nil
				return
			}
		case <-ctx.Done():
			resp.State, resp.err = reply, QuorumCallError{ctx.Err().Error(), len(replyValues), errs}
			return
		}

		if len(errs)+len(replyValues) == expected {
			resp.State, resp.err = reply, QuorumCallError{"incomplete call", len(replyValues), errs}
			return
		}
	}
}

func callGRPCReadFuture(ctx context.Context, node *Node, arg *ReadRequest, replyChan chan<- internalState) {
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
	replyChan <- internalState{node.id, reply, err}
}

/* Exported asynchronous quorum call method WriteFuture */

// WriteFuture asynchronously invokes a quorum call on configuration c
// and returns a FutureWriteResponse which can be used to inspect the quorum call
// reply and error when available.
func (c *Configuration) WriteFuture(ctx context.Context, arg *State) *FutureWriteResponse {
	f := &FutureWriteResponse{
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
func (f *FutureWriteResponse) Get() (*WriteResponse, error) {
	<-f.c
	return f.WriteResponse, f.err
}

// Done reports if a reply and/or error is available for the WriteFuture.
func (f *FutureWriteResponse) Done() bool {
	select {
	case <-f.c:
		return true
	default:
		return false
	}
}

/* Unexported asynchronous quorum call method WriteFuture */

func (c *Configuration) writeFuture(ctx context.Context, a *State, resp *FutureWriteResponse) {
	var ti traceInfo
	if c.mgr.opts.trace {
		ti.Trace = trace.New("gorums."+c.tstring()+".Sent", "WriteFuture")
		defer ti.Finish()

		ti.firstLine.cid = c.id
		if deadline, ok := ctx.Deadline(); ok {
			ti.firstLine.deadline = time.Until(deadline)
		}
		ti.LazyLog(&ti.firstLine, false)
		ti.LazyLog(&payload{sent: true, msg: a}, false)

		defer func() {
			ti.LazyLog(&qcresult{
				ids:   resp.NodeIDs,
				reply: resp.WriteResponse,
				err:   resp.err,
			}, false)
			if resp.err != nil {
				ti.SetError()
			}
		}()
	}

	expected := c.n
	replyChan := make(chan internalWriteResponse, expected)
	for _, n := range c.nodes {
		go callGRPCWriteFuture(ctx, n, a, replyChan)
	}

	var (
		replyValues = make([]*WriteResponse, 0, c.n)
		reply       *WriteResponse
		errs        []GRPCError
		quorum      bool
	)

	for {
		select {
		case r := <-replyChan:
			resp.NodeIDs = append(resp.NodeIDs, r.nid)
			if r.err != nil {
				errs = append(errs, GRPCError{r.nid, r.err})
				break
			}
			if c.mgr.opts.trace {
				ti.LazyLog(&payload{sent: false, id: r.nid, msg: r.reply}, false)
			}
			replyValues = append(replyValues, r.reply)
			if reply, quorum = c.qspec.WriteFutureQF(a, replyValues); quorum {
				resp.WriteResponse, resp.err = reply, nil
				return
			}
		case <-ctx.Done():
			resp.WriteResponse, resp.err = reply, QuorumCallError{ctx.Err().Error(), len(replyValues), errs}
			return
		}

		if len(errs)+len(replyValues) == expected {
			resp.WriteResponse, resp.err = reply, QuorumCallError{"incomplete call", len(replyValues), errs}
			return
		}
	}
}

func callGRPCWriteFuture(ctx context.Context, node *Node, arg *State, replyChan chan<- internalWriteResponse) {
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
	replyChan <- internalWriteResponse{node.id, reply, err}
}
