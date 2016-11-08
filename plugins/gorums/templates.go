// DO NOT EDIT. Generated by github.com/relab/gorums/cmd/gentemplates
// Template source files to edit is in the: 'dev' folder

package gorums

const config_rpc_tmpl = `
{{/* Remember to run 'make gengolden' after editing this file. */}}

{{- if not .IgnoreImports}}
package {{.PackageName}}

import (
	"fmt"

	"golang.org/x/net/context"
)

{{- end}}

{{range $elm := .Services}}

{{if .Multicast}}

// {{.MethodName}} invokes an asynchronous {{.MethodName}} RPC on configuration c.
// The call has no return value and is invoked on every node in the
// configuration.
func (c *Configuration) {{.MethodName}}(ctx context.Context, args *{{.ReqName}}) error {
	return c.mgr.{{.UnexportedMethodName}}(ctx, c, args)
}

{{else -}}

// {{.TypeName}} encapsulates the reply from a {{.MethodName}} RPC invocation.
// It contains the id of each node in the quorum that replied and a single
// reply.
type {{.TypeName}} struct {
	NodeIDs []uint32
	*{{.RespName}}
}

func (r {{.TypeName}}) String() string {
	return fmt.Sprintf("node ids: %v | answer: %v", r.NodeIDs, r.{{.RespName}})
}

// {{.MethodName}} invokes a {{.MethodName}} RPC on configuration c
// and returns the result as a {{.TypeName}}.
func (c *Configuration) {{.MethodName}}(ctx context.Context, args *{{.ReqName}}) (*{{.TypeName}}, error) {
	return c.mgr.{{.UnexportedMethodName}}(ctx, c, args)
}

{{if .GenFuture}}

// {{.MethodName}}Future is a reference to an asynchronous {{.MethodName}} RPC invocation.
type {{.MethodName}}Future struct {
	reply *{{.TypeName}}
	err   error
	c     chan struct{}
}

// {{.MethodName}}Future asynchronously invokes a {{.MethodName}} RPC on configuration c and
// returns a {{.MethodName}}Future which can be used to inspect the RPC reply and error
// when available.
func (c *Configuration) {{.MethodName}}Future(ctx context.Context, args *{{.ReqName}}) *{{.MethodName}}Future {
	f := new({{.MethodName}}Future)
	f.c = make(chan struct{}, 1)
	go func() {
		defer close(f.c)
		f.reply, f.err = c.mgr.{{.UnexportedMethodName}}(ctx, c, args)
	}()
	return f
}

// Get returns the reply and any error associated with the {{.MethodName}}Future.
// The method blocks until a reply or error is available.
func (f *{{.MethodName}}Future) Get() (*{{.TypeName}}, error) {
	<-f.c
	return f.reply, f.err
}

// Done reports if a reply or error is available for the {{.MethodName}}Future.
func (f *{{.MethodName}}Future) Done() bool {
	select {
	case <-f.c:
		return true
	default:
		return false
	}
}

{{- end -}}
{{- end -}}
{{- end -}}
`

const mgr_rpc_tmpl = `
{{/* Remember to run 'make gengolden' after editing this file. */}}
{{$pkgName := .PackageName}}

{{if not .IgnoreImports}}
package {{$pkgName}}

import (
	"time"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)
{{end}}

{{range $elm := .Services}}

{{if .Multicast}}
func (m *Manager) {{.UnexportedMethodName}}(ctx context.Context, c *Configuration, args *{{.ReqName}}) error {
	for _, node := range c.nodes {
		go func(n *Node) {
			err := n.{{.MethodName}}Client.Send(args)
			if err == nil {
				return
			}
			if m.logger != nil {
				m.logger.Printf("%d: {{.UnexportedMethodName}} stream send error: %v", n.id, err)
			}
		}(node)
	}

	return nil
}

{{else}}

type {{.UnexportedTypeName}} struct {
	nid   uint32
	reply *{{.RespName}}
	err   error
}

func (m *Manager) {{.UnexportedMethodName}}(ctx context.Context, c *Configuration, args *{{.ReqName}}) (*{{.TypeName}}, error) {
	replyChan := make(chan {{.UnexportedTypeName}}, c.n)
	newCtx, cancel := context.WithCancel(ctx)

	for _, n := range c.nodes {
		go callGRPC{{.MethodName}}(newCtx, n, args, replyChan)
	}

	var (
		replyValues = make([]*{{.RespName}}, 0, c.n)
		reply       = &{{.TypeName}}{NodeIDs: make([]uint32, 0, c.n)}
		errCount    int
		quorum      bool
	)

	for {
		select {
		case r := <-replyChan:
			if r.err != nil {
				errCount++
				break
			}
			replyValues = append(replyValues, r.reply)
			reply.NodeIDs = append(reply.NodeIDs, r.nid)
			if reply.{{.RespName}}, quorum = c.qspec.{{.MethodName}}QF(replyValues); quorum {
				cancel()
				return reply, nil
			}
		case <-newCtx.Done():
			return reply, QuorumCallError{ctx.Err().Error(), errCount, len(replyValues)}
		}

		if errCount+len(replyValues) == c.n {
			cancel()
			return reply, QuorumCallError{"incomplete call", errCount, len(replyValues)}
		}
	}
}

func callGRPC{{.MethodName}}(ctx context.Context, node *Node, args *{{.ReqName}}, replyChan chan<- {{.UnexportedTypeName}}) {
	reply := new({{.RespName}})
	start := time.Now()
	err := grpc.Invoke(
		ctx,
		"/{{$pkgName}}.{{.ServName}}/{{.MethodName}}",
		args,
		reply,
		node.conn,
	)
	switch grpc.Code(err) { // nil -> codes.OK
	case codes.OK, codes.Canceled:
		node.setLatency(time.Since(start))
	default:
		node.setLastErr(err)
	}
	replyChan <- {{.UnexportedTypeName}}{node.id, reply, err}
}

{{- end -}}
{{- end -}}
`

const node_tmpl = `
{{/* Remember to run 'make gengolden' after editing this file. */}}

{{- if not .IgnoreImports}}
package {{.PackageName}}

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
)
{{- end}}

// Node encapsulates the state of a node on which a remote procedure call
// can be made.
type Node struct {
	// Only assigned at creation.
	id   uint32
	self bool
	addr string
	conn *grpc.ClientConn


{{range .Clients}}
	{{.}} {{.}}
{{end}}

{{range .Services}}
{{if .Multicast}}
	{{.MethodName}}Client {{.ServName}}_{{.MethodName}}Client
{{end}}
{{end}}

	sync.Mutex
	lastErr error
	latency time.Duration
}

func (n *Node) connect(opts ...grpc.DialOption) error {
  	var err error
	n.conn, err = grpc.Dial(n.addr, opts...)
	if err != nil {
		return fmt.Errorf("dialing node failed: %v", err)
	}

{{range .Clients}}
	n.{{.}} = New{{.}}(n.conn)
{{end}}

{{range .Services}}
{{if .Multicast}}
  	n.{{.MethodName}}Client, err = n.{{.ServName}}Client.{{.MethodName}}(context.Background())
  	if err != nil {
  		return fmt.Errorf("stream creation failed: %v", err)
  	}
{{end}}
{{end -}}

	return nil
}

func (n *Node) close() error {
	// TODO: Log error, mainly care about the connection error below.
        // We should log this error, but we currently don't have access to the
        // logger in the manager.
{{- range .Services -}}
{{if .Multicast}}
	_, _ = n.{{.MethodName}}Client.CloseAndRecv()
{{- end -}}
{{end}}
	
	if err := n.conn.Close(); err != nil {
                return fmt.Errorf("conn close error: %v", err)
        }	
	return nil
}
`

const qspec_tmpl = `
{{/* Remember to run 'make gengolden' after editing this file. */}}

{{- if not .IgnoreImports}}
package {{.PackageName}}
{{- end}}

// QuorumSpec is the interface that wraps every quorum function.
type QuorumSpec interface {
{{- range $elm := .Services}}
{{- if not .Multicast}}
	// {{.MethodName}}QF is the quorum function for the {{.MethodName}} RPC method.
	{{.MethodName}}QF(replies []*{{.RespName}}) (*{{.RespName}}, bool)
{{- end -}}
{{- end}}
}
`

var templates = map[string]string{
	"config_rpc_tmpl": config_rpc_tmpl,
	"mgr_rpc_tmpl":    mgr_rpc_tmpl,
	"node_tmpl":       node_tmpl,
	"qspec_tmpl":      qspec_tmpl,
}
