package method

import (
	"context"
	"errors"
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/spaolacci/murmur3"
	"log"
	"sort"
	"strconv"
	"sync"
)

const numberOfReplicas = 500

var (
	PgwNodeRing    *ConsistentHashNodeRing
	NodeUpdateChan = make(chan []string, 1)
	ErrEmptyCircle = errors.New("empty circle")
)

// 一致性哈希环,用于管理服务器节点.
type ConsistentHashNodeRing struct {
	ring *Consistent
	sync.RWMutex
}
type client struct {
	consul *consul.Client
}

type Client interface {
	// Get a Service from consul
	GetService(string, string) ([]string, error)
	// register a service with local agent
	ServiceRegister(string, string, int) error
	// Deregister a service with local agent
	DeRegister(string) error
}

func NewConsulClient(addr string) (*client, error) {
	config := consul.DefaultConfig()
	config.Address = addr
	c, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &client{consul: c}, nil
}

// Register a service with consul local agent
func (c *client) ServiceRegister(srvName, srvHost string, srvPort int) error {
	reg := new(consul.AgentServiceRegistration)
	reg.Name = srvName
	thisId := fmt.Sprintf("%s_%d", srvHost, srvPort)
	reg.ID = thisId
	reg.Port = srvPort
	reg.Address = srvHost
	//增加check
	check := new(consul.AgentServiceCheck)
	check.HTTP = fmt.Sprintf("http://%s:%d%s", reg.Address, reg.Port, "/-/healthy")
	//设置超时 5s。
	check.Timeout = "2s"
	check.DeregisterCriticalServiceAfter = "5s"
	//设置间隔 5s。
	check.Interval = "5s"
	//注册check服务。
	reg.Check = check

	return c.consul.Agent().ServiceRegister(reg)
}

// DeRegister a service with consul local agent
func (c *client) DeRegister(id string) error {
	return c.consul.Agent().ServiceDeregister(id)
}

func RegisterFromFile(c *client, servers []string, srvName string, srvPort int) (errors []error) {

	for _, addr := range servers {

		e := c.ServiceRegister(srvName, addr, srvPort)
		if e != nil {
			errors = append(errors, e)
		}

	}
	return
}
func (c *client) RunRefreshServiceNode(ctx context.Context, srvName string, consulServerAddr string) error {
	go RunReshardHashRing(ctx)

	errchan := make(chan error, 1)
	go func() {
		errchan <- c.WatchService(ctx, srvName, consulServerAddr)

	}()
	select {
	case <-ctx.Done():
		return nil
	case err := <-errchan:
		return err
	}
	return nil
}

func (c *client) WatchService(ctx context.Context, srvName string, consulServerAddr string) error {

	watchConfig := make(map[string]interface{})

	watchConfig["type"] = "service"
	watchConfig["service"] = srvName
	watchConfig["handler_type"] = "script"
	watchConfig["passingonly"] = true
	watchPlan, err := watch.Parse(watchConfig)
	if err != nil {
		return err

	}

	watchPlan.Handler = func(lastIndex uint64, result interface{}) {
		if entries, ok := result.([]*consul.ServiceEntry); ok {
			var hs []string

			for _, a := range entries {

				hs = append(hs, fmt.Sprintf("%s:%d", a.Service.Address, a.Service.Port))
			}
			if len(hs) > 0 {
				NodeUpdateChan <- hs
			}

		}

	}
	if err := watchPlan.Run(consulServerAddr); err != nil {
		return err
	}
	return nil

}

func NewConsistentHashNodesRing(nodes []string) *ConsistentHashNodeRing {
	ret := &ConsistentHashNodeRing{ring: New()}

	ret.SetNumberOfReplicas(numberOfReplicas)
	ret.SetNodes(nodes)
	PgwNodeRing = ret
	return ret
}

func (this *ConsistentHashNodeRing) ReShardRing(nodes []string) {
	this.Lock()
	defer this.Unlock()
	newRing := New()
	newRing.NumberOfReplicas = numberOfReplicas
	for _, node := range nodes {
		newRing.Add(node)
	}
	this.ring = newRing
}

// 根据pk,获取node节点. chash(pk) -> node
func (this *ConsistentHashNodeRing) GetNode(pk string) (string, error) {
	this.RLock()
	defer this.RUnlock()

	return this.ring.Get(pk)
}

func (this *ConsistentHashNodeRing) SetNodes(nodes []string) {
	for _, node := range nodes {
		this.ring.Add(node)
	}
}

func (this *ConsistentHashNodeRing) SetNumberOfReplicas(num int32) {
	this.ring.NumberOfReplicas = int(num)
}

func StringSliceEqualBCE(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	b = b[:len(a)]
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func RunReshardHashRing(ctx context.Context) {

	for {
		select {
		case nodes := <-NodeUpdateChan:

			oldNodes := PgwNodeRing.ring.Members()
			sort.Strings(nodes)
			sort.Strings(oldNodes)
			isEq := StringSliceEqualBCE(nodes, oldNodes)
			if isEq == false {
				PgwNodeRing.ReShardRing(nodes)
			} else {
				log.Print("nodes is equal")
			}
		case <-ctx.Done():
			return
		}

	}
}

type uints []uint32

// Len returns the length of the uints array.
func (x uints) Len() int { return len(x) }

// Less returns true if element i is less than element j.
func (x uints) Less(i, j int) bool { return x[i] < x[j] }

// Swap exchanges elements i and j.
func (x uints) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// ErrEmptyCircle is the error returned when trying to get an element when nothing has been added to hash.

// Consistent holds the information about the members of the consistent hash circle.
type Consistent struct {
	circle           map[uint32]string
	members          map[string]bool
	sortedHashes     uints
	NumberOfReplicas int
	count            int64
	scratch          [64]byte
	sync.RWMutex
}

// New creates a new Consistent object with a default setting of 20 replicas for each entry.
//
// To change the number of replicas, set NumberOfReplicas before adding entries.
func New() *Consistent {
	c := new(Consistent)
	c.NumberOfReplicas = 20
	c.circle = make(map[uint32]string)
	c.members = make(map[string]bool)
	return c
}

// eltKey generates a string key for an element with an index.
func (c *Consistent) eltKey(elt string, idx int) string {
	// return elt + "|" + strconv.Itoa(idx)
	return strconv.Itoa(idx) + elt
}

// Add inserts a string element in the consistent hash.
func (c *Consistent) Add(elt string) {
	c.Lock()
	defer c.Unlock()
	c.add(elt)
}

// need c.Lock() before calling
func (c *Consistent) add(elt string) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		c.circle[c.hashKey(c.eltKey(elt, i))] = elt
	}
	c.members[elt] = true
	c.updateSortedHashes()
	c.count++
}

// Remove removes an element from the hash.
func (c *Consistent) Remove(elt string) {
	c.Lock()
	defer c.Unlock()
	c.remove(elt)
}

// need c.Lock() before calling
func (c *Consistent) remove(elt string) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		delete(c.circle, c.hashKey(c.eltKey(elt, i)))
	}
	delete(c.members, elt)
	c.updateSortedHashes()
	c.count--
}

// Set sets all the elements in the hash.  If there are existing elements not
// present in elts, they will be removed.
func (c *Consistent) Set(elts []string) {
	c.Lock()
	defer c.Unlock()
	for k := range c.members {
		found := false
		for _, v := range elts {
			if k == v {
				found = true
				break
			}
		}
		if !found {
			c.remove(k)
		}
	}
	for _, v := range elts {
		_, exists := c.members[v]
		if exists {
			continue
		}
		c.add(v)
	}
}

func (c *Consistent) Members() []string {
	c.RLock()
	defer c.RUnlock()
	var m []string
	for k := range c.members {
		m = append(m, k)
	}
	return m
}

// Get returns an element close to where name hashes to in the circle.
func (c *Consistent) Get(name string) (string, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return "", ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	return c.circle[c.sortedHashes[i]], nil
}

func (c *Consistent) search(key uint32) (i int) {
	f := func(x int) bool {
		return c.sortedHashes[x] > key
	}
	i = sort.Search(len(c.sortedHashes), f)
	if i >= len(c.sortedHashes) {
		i = 0
	}
	return
}

// GetTwo returns the two closest distinct elements to the name input in the circle.
func (c *Consistent) GetTwo(name string) (string, string, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return "", "", ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	a := c.circle[c.sortedHashes[i]]

	if c.count == 1 {
		return a, "", nil
	}

	start := i
	var b string
	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		b = c.circle[c.sortedHashes[i]]
		if b != a {
			break
		}
	}
	return a, b, nil
}

// GetN returns the N closest distinct elements to the name input in the circle.
func (c *Consistent) GetN(name string, n int) ([]string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.circle) == 0 {
		return nil, ErrEmptyCircle
	}

	if c.count < int64(n) {
		n = int(c.count)
	}

	var (
		key   = c.hashKey(name)
		i     = c.search(key)
		start = i
		res   = make([]string, 0, n)
		elem  = c.circle[c.sortedHashes[i]]
	)

	res = append(res, elem)

	if len(res) == n {
		return res, nil
	}

	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		elem = c.circle[c.sortedHashes[i]]
		if !sliceContainsMember(res, elem) {
			res = append(res, elem)
		}
		if len(res) == n {
			break
		}
	}

	return res, nil
}

func (c *Consistent) hashKey(key string) uint32 {
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		//return crc32.ChecksumIEEE(scratch[:len(key)])
		return murmur3.Sum32(scratch[:len(key)])
	}
	//return crc32.ChecksumIEEE([]byte(key))
	return murmur3.Sum32([]byte(key))
}

func (c *Consistent) updateSortedHashes() {
	hashes := c.sortedHashes[:0]
	//reallocate if we're holding on to too much (1/4th)
	if cap(c.sortedHashes)/(c.NumberOfReplicas*4) > len(c.circle) {
		hashes = nil
	}
	for k := range c.circle {
		hashes = append(hashes, k)
	}
	sort.Sort(hashes)
	c.sortedHashes = hashes
}

func sliceContainsMember(set []string, member string) bool {
	for _, m := range set {
		if m == member {
			return true
		}
	}
	return false
}
