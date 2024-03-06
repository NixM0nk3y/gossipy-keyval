package cluster

import (
	"encoding/json"
	"gossipy/internal/store"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/rs/zerolog/log"
)

type Node struct {
	Name string
	Addr string
	Port int
}

type BroadcastKVOperation struct {
	Operation string `json:"operation"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func (m BroadcastKVOperation) Invalidates(other memberlist.Broadcast) bool {
	return false
}
func (m BroadcastKVOperation) Finished() {
	// nop
}
func (m BroadcastKVOperation) Message() []byte {
	data, err := json.Marshal(m)
	if err != nil {
		return []byte("")
	}
	return data
}

func ParseMyBroadcastMessage(data []byte) (*BroadcastKVOperation, bool) {
	msg := new(BroadcastKVOperation)
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, false
	}
	return msg, true
}

//
// https://github.com/octu0/example-memberlist/blob/master/05-broadcast/c.go
//

type NodeEventDelegate struct {
	Num int
}

func (d *NodeEventDelegate) NotifyJoin(node *memberlist.Node) {
	d.Num += 1
}
func (d *NodeEventDelegate) NotifyLeave(node *memberlist.Node) {
	d.Num -= 1
}
func (d *NodeEventDelegate) NotifyUpdate(node *memberlist.Node) {
	// skip
}

type NodeDelegate struct {
	msgCh      chan []byte
	broadcasts *memberlist.TransmitLimitedQueue
}

func (d *NodeDelegate) NotifyMsg(msg []byte) {
	if len(msg) == 0 {
		return
	}
	d.msgCh <- msg
}
func (d *NodeDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}
func (d *NodeDelegate) NodeMeta(limit int) []byte {
	// not use, noop
	return []byte("")
}
func (d *NodeDelegate) LocalState(join bool) []byte {
	// not use, noop
	return []byte("")
}
func (d *NodeDelegate) MergeRemoteState(buf []byte, join bool) {
	// not use
}

type Cluster struct {
	*memberlist.Memberlist
	LocalNode  *Node
	Broadcasts *memberlist.TransmitLimitedQueue
}

func NewCluster(localNode *Node, store *store.KeyValueStore, msgChannel chan []byte) (*Cluster, error) {
	log.Debug().Msg("NewCluster")

	e := &NodeEventDelegate{}
	e.Num = 0

	d := &NodeDelegate{}
	d.msgCh = msgChannel
	d.broadcasts = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			log.Info().Msgf("broadcast nodes = %d", e.Num)
			return e.Num
		},
		RetransmitMult: 3,
	}

	config := memberlist.DefaultLocalConfig()
	config.Name = localNode.Name
	config.BindAddr = localNode.Addr
	config.BindPort = localNode.Port
	config.AdvertisePort = config.BindPort
	config.Events = e
	config.Delegate = d

	log.Info().Str("name", config.Name).Str("address", localNode.Addr).Msg("creating cluster")

	list, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		Memberlist: list,
		LocalNode:  localNode,
		Broadcasts: d.broadcasts,
	}, nil
}

func (c *Cluster) Join(seeds []string) error {
	log.Info().Msgf("joining cluster: %v", seeds)
	_, err := c.Memberlist.Join(seeds)
	return err
}

func (c *Cluster) Leave(timeout time.Duration) error {
	log.Info().Msg("leaving cluster")
	err := c.Memberlist.Leave(timeout)
	return err
}

func (c *Cluster) NotifyMsg(msg []byte) {
	log.Info().Msg("NotifyMsg")
}
