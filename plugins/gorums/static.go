// DO NOT EDIT. Generated by github.com/relab/gorums/cmd/bundle
// bundle github.com/relab/gorums/dev

package gorums

var staticImports = []string{
	"fmt",
	"google.golang.org/grpc",
	"time",
	"sort",
	"sync",
	"net",
	"log",
	"hash/fnv",
	"encoding/binary",
}

const staticResources = `
/* config.go */

// A Configuration represents a static set of nodes on which quorum remote
// procedure calls may be invoked.
type Configuration struct {
	id	uint32
	nodes	[]*Node
	n	int
	mgr	*Manager
	qspec	QuorumSpec
}

// ID reports the identifier for the configuration.
func (c *Configuration) ID() uint32 {
	return c.id
}

// NodeIDs returns a slice containing the local ids of all the nodes in the
// configuration.
func (c *Configuration) NodeIDs() []uint32 {
	ids := make([]uint32, len(c.nodes))
	for i, node := range c.nodes {
		ids[i] = node.ID()
	}
	return ids
}

// Size returns the number of nodes in the configuration.
func (c *Configuration) Size() int {
	return c.n
}

func (c *Configuration) String() string {
	return fmt.Sprintf("configuration %d", c.id)
}

// Equal returns a boolean reporting whether a and b represents the same
// configuration.
func Equal(a, b *Configuration) bool	{ return a.id == b.id }

// NewTestConfiguration returns a new configuration with quorum size q and
// node size n. No other fields are set. Configurations returned from this
// constructor should only be used when testing quorum functions.
func NewTestConfiguration(q, n int) *Configuration {
	return &Configuration{
		nodes: make([]*Node, n),
	}
}

/* errors.go */

// A NodeNotFoundError reports that a specified node could not be found.
type NodeNotFoundError uint32

func (e NodeNotFoundError) Error() string {
	return fmt.Sprintf("node not found: %d", e)
}

// A ConfigNotFoundError reports that a specified configuration could not be
// found.
type ConfigNotFoundError uint32

func (e ConfigNotFoundError) Error() string {
	return fmt.Sprintf("configuration not found: %d", e)
}

// An IllegalConfigError reports that a specified configuration could not be
// created.
type IllegalConfigError string

func (e IllegalConfigError) Error() string {
	return "illegal configuration: " + string(e)
}

// ManagerCreationError returns an error reporting that a Manager could not be
// created due to err.
func ManagerCreationError(err error) error {
	return fmt.Errorf("could not create manager: %s", err.Error())
}

// A QuorumCallError is used to report that a quorum call failed.
type QuorumCallError struct {
	Reason			string
	ErrCount, ReplyCount	int
}

func (e QuorumCallError) Error() string {
	return fmt.Sprintf(
		"quorum call error: %s (errors: %d, replies: %d)",
		e.Reason, e.ErrCount, e.ReplyCount,
	)
}

/* level.go */

// LevelNotSet is the zero value level used to indicate that no level (and
// thereby no reply) has been set for a correctable quorum call.
const LevelNotSet = -1

/* mgr.go */

// Manager manages a pool of node configurations on which quorum remote
// procedure calls can be made.
type Manager struct {
	sync.RWMutex

	nodes	map[uint32]*Node
	configs	map[uint32]*Configuration

	closeOnce	sync.Once
	logger		*log.Logger
	opts		managerOptions
}

// NewManager attempts to connect to the given set of node addresses and if
// successful returns a new Manager containing connections to those nodes.
func NewManager(nodeAddrs []string, opts ...ManagerOption) (*Manager, error) {
	if len(nodeAddrs) == 0 {
		return nil, fmt.Errorf("could not create manager: no nodes provided")
	}

	m := &Manager{
		nodes:		make(map[uint32]*Node),
		configs:	make(map[uint32]*Configuration),
	}

	for _, opt := range opts {
		opt(&m.opts)
	}

	selfAddrIndex, selfID, err := m.parseSelfOptions(nodeAddrs)
	if err != nil {
		return nil, ManagerCreationError(err)
	}

	idSeen := false
	for i, naddr := range nodeAddrs {
		node, err2 := m.createNode(naddr)
		if err2 != nil {
			return nil, ManagerCreationError(err2)
		}
		m.nodes[node.id] = node
		if i == selfAddrIndex {
			node.self = true
			continue
		}
		if node.id == selfID {
			node.self = true
			idSeen = true
		}
	}
	if selfID != 0 && !idSeen {
		return nil, ManagerCreationError(
			fmt.Errorf("WithSelfID provided, but no node with id %d found", selfID),
		)
	}

	err = m.connectAll()
	if err != nil {
		return nil, ManagerCreationError(err)
	}

	if m.opts.logger != nil {
		m.logger = m.opts.logger
	}

	return m, nil
}

func (m *Manager) parseSelfOptions(addrs []string) (int, uint32, error) {
	if m.opts.selfAddr != "" && m.opts.selfID != 0 {
		return 0, 0, fmt.Errorf("both WithSelfAddr and WithSelfID provided")
	}
	if m.opts.selfID != 0 {
		return -1, m.opts.selfID, nil
	}
	if m.opts.selfAddr == "" {
		return -1, 0, nil
	}

	seen, index := contains(m.opts.selfAddr, addrs)
	if !seen {
		return 0, 0, fmt.Errorf(
			"option WithSelfAddr provided, but address %q was not present in address list",
			m.opts.selfAddr)
	}

	return index, 0, nil
}

func (m *Manager) createNode(addr string) (*Node, error) {
	m.Lock()
	defer m.Unlock()

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("create node %s error: %v", addr, err)
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(tcpAddr.String()))
	id := h.Sum32()

	if _, found := m.nodes[id]; found {
		return nil, fmt.Errorf("create node %s error: node already exists", addr)
	}

	node := &Node{
		id:		id,
		addr:		tcpAddr.String(),
		latency:	-1 * time.Second,
	}

	return node, nil
}

func (m *Manager) connectAll() error {
	if m.opts.noConnect {
		return nil
	}
	for _, node := range m.nodes {
		if node.self {
			continue
		}
		err := node.connect(m.opts.grpcDialOpts...)
		if err != nil {
			return fmt.Errorf("connect node %s error: %v", node.addr, err)
		}
	}
	return nil
}

func (m *Manager) closeNodeConns() {
	for _, node := range m.nodes {
		if node.self {
			continue
		}
		err := node.close()
		if err == nil {
			continue
		}
		if m.logger != nil {
			m.logger.Printf("node %d: error closing: %v", node.id, err)
		}
	}
}

// Close closes all node connections and any client streams.
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		m.closeNodeConns()
	})
}

// NodeIDs returns the identifier of each available node.
func (m *Manager) NodeIDs() []uint32 {
	m.RLock()
	defer m.RUnlock()
	ids := make([]uint32, 0, len(m.nodes))
	for id := range m.nodes {
		ids = append(ids, id)
	}
	sort.Sort(idSlice(ids))
	return ids
}

// Node returns the node with the given identifier if present.
func (m *Manager) Node(id uint32) (node *Node, found bool) {
	m.RLock()
	defer m.RUnlock()
	node, found = m.nodes[id]
	return node, found
}

// Nodes returns a slice of each available node.
func (m *Manager) Nodes(excludeSelf bool) []*Node {
	m.RLock()
	defer m.RUnlock()
	var nodes []*Node
	for _, node := range m.nodes {
		if excludeSelf && node.self {
			continue
		}
		nodes = append(nodes, node)
	}
	OrderedBy(ID).Sort(nodes)
	return nodes
}

// ConfigurationIDs returns the identifier of each available
// configuration.
func (m *Manager) ConfigurationIDs() []uint32 {
	m.RLock()
	defer m.RUnlock()
	ids := make([]uint32, 0, len(m.configs))
	for id := range m.configs {
		ids = append(ids, id)
	}
	sort.Sort(idSlice(ids))
	return ids
}

// Configuration returns the configuration with the given global
// identifier if present.
func (m *Manager) Configuration(id uint32) (config *Configuration, found bool) {
	m.RLock()
	defer m.RUnlock()
	config, found = m.configs[id]
	return config, found
}

// Configurations returns a slice of each available configuration.
func (m *Manager) Configurations() []*Configuration {
	m.RLock()
	defer m.RUnlock()
	configs := make([]*Configuration, 0, len(m.configs))
	for _, conf := range m.configs {
		configs = append(configs, conf)
	}
	return configs
}

// Size returns the number of nodes and configurations in the Manager.
func (m *Manager) Size() (nodes, configs int) {
	m.RLock()
	defer m.RUnlock()
	return len(m.nodes), len(m.configs)
}

// AddNode attempts to dial to the provide node address. The node is
// added to the Manager's pool of nodes if a connection was established.
func (m *Manager) AddNode(addr string) error {
	panic("not implemented")
}

// NewConfiguration returns a new configuration given quorum specification and
// a timeout.
func (m *Manager) NewConfiguration(ids []uint32, qspec QuorumSpec) (*Configuration, error) {
	m.Lock()
	defer m.Unlock()

	if len(ids) == 0 {
		return nil, IllegalConfigError("need at least one node")
	}

	var cnodes []*Node
	for _, nid := range ids {
		node, found := m.nodes[nid]
		if !found {
			return nil, NodeNotFoundError(nid)
		}
		if node.self && m.selfSpecified() {
			return nil, IllegalConfigError(
				fmt.Sprintf("self (%d) can't be part of a configuration when a self-option is provided", nid),
			)
		}
		cnodes = append(cnodes, node)
	}

	// Node ids are sorted ensure a globally consistent configuration id.
	OrderedBy(ID).Sort(cnodes)

	h := fnv.New32a()
	for _, node := range cnodes {
		binary.Write(h, binary.LittleEndian, node.id)
	}
	cid := h.Sum32()

	conf, found := m.configs[cid]
	if found {
		return conf, nil
	}

	c := &Configuration{
		id:	cid,
		nodes:	cnodes,
		n:	len(cnodes),
		mgr:	m,
		qspec:	qspec,
	}
	m.configs[cid] = c

	return c, nil
}

func (m *Manager) selfSpecified() bool {
	return m.opts.selfAddr != "" || m.opts.selfID != 0
}

type idSlice []uint32

func (p idSlice) Len() int		{ return len(p) }
func (p idSlice) Less(i, j int) bool	{ return p[i] < p[j] }
func (p idSlice) Swap(i, j int)		{ p[i], p[j] = p[j], p[i] }

/* node_func.go */

// ID returns the ID of m.
func (n *Node) ID() uint32 {
	return n.id
}

// Address returns network address of m.
func (n *Node) Address() string {
	return n.addr
}

func (n *Node) String() string {
	n.Lock()
	defer n.Unlock()
	return fmt.Sprintf(
		"node %d | addr: %s | latency: %v",
		n.id, n.addr, n.latency,
	)
}

func (n *Node) setLastErr(err error) {
	n.Lock()
	defer n.Unlock()
	n.lastErr = err
}

// LastErr returns the last error encountered (if any) when invoking a remote
// procedure call on this node.
func (n *Node) LastErr() error {
	n.Lock()
	defer n.Unlock()
	return n.lastErr
}

func (n *Node) setLatency(lat time.Duration) {
	n.Lock()
	defer n.Unlock()
	n.latency = lat
}

// Latency returns the latency of the last successful remote procedure call
// made to this node.
func (n *Node) Latency() time.Duration {
	n.Lock()
	defer n.Unlock()
	return n.latency
}

type lessFunc func(n1, n2 *Node) bool

// MultiSorter implements the Sort interface, sorting the nodes within.
type MultiSorter struct {
	nodes	[]*Node
	less	[]lessFunc
}

// Sort sorts the argument slice according to the less functions passed to
// OrderedBy.
func (ms *MultiSorter) Sort(nodes []*Node) {
	ms.nodes = nodes
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *MultiSorter {
	return &MultiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *MultiSorter) Len() int {
	return len(ms.nodes)
}

// Swap is part of sort.Interface.
func (ms *MultiSorter) Swap(i, j int) {
	ms.nodes[i], ms.nodes[j] = ms.nodes[j], ms.nodes[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that is either Less or
// !Less. Note that it can call the less functions twice per call. We
// could change the functions to return -1, 0, 1 and reduce the
// number of calls for greater efficiency: an exercise for the reader.
func (ms *MultiSorter) Less(i, j int) bool {
	p, q := ms.nodes[i], ms.nodes[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}

// ID sorts nodes by their identifier in increasing order.
var ID = func(n1, n2 *Node) bool {
	return n1.id < n2.id
}

// Latency sorts nodes by latency in increasing order. Latencies less then
// zero (sentinel value) are considered greater than any positive latency.
var Latency = func(n1, n2 *Node) bool {
	if n1.latency < 0 {
		return false
	}
	return n1.latency < n2.latency

}

// Error sorts nodes by their LastErr() status in increasing order. A
// node with LastErr() != nil is larger than a node with LastErr() == nil.
var Error = func(n1, n2 *Node) bool {
	if n1.lastErr != nil && n2.lastErr == nil {
		return false
	}
	return true
}

/* opts.go */

type managerOptions struct {
	grpcDialOpts	[]grpc.DialOption
	logger		*log.Logger
	noConnect	bool
	selfAddr	string
	selfID		uint32
}

// ManagerOption provides a way to set different options on a new Manager.
type ManagerOption func(*managerOptions)

// WithGrpcDialOptions returns a ManagerOption which sets any gRPC dial options
// the Manager should use when initially connecting to each node in its
// pool.
func WithGrpcDialOptions(opts ...grpc.DialOption) ManagerOption {
	return func(o *managerOptions) {
		o.grpcDialOpts = opts
	}
}

// WithLogger returns a ManagerOption which sets an optional error logger for
// the Manager.
func WithLogger(logger *log.Logger) ManagerOption {
	return func(o *managerOptions) {
		o.logger = logger
	}
}

// WithNoConnect returns a ManagerOption which instructs the Manager not to
// connect to any of its nodes. Mainly used for testing purposes.
func WithNoConnect() ManagerOption {
	return func(o *managerOptions) {
		o.noConnect = true
	}
}

// WithSelfAddr returns a ManagerOption which instructs the Manager not to connect
// to the node with network address addr. The address must be present in the
// list of node addresses provided to the Manager.
func WithSelfAddr(addr string) ManagerOption {
	return func(o *managerOptions) {
		o.selfAddr = addr
	}
}

// WithSelfID returns a ManagerOption which instructs the Manager not to
// connect to the node with the given id. The node must be present in the list
// of node addresses provided to the Manager.
func WithSelfID(id uint32) ManagerOption {
	return func(o *managerOptions) {
		o.selfID = id
	}
}

/* util.go */

func contains(addr string, addrs []string) (found bool, index int) {
	for i, a := range addrs {
		if addr == a {
			return true, i
		}
	}
	return false, -1
}
`
