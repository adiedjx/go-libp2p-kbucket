//go:generate go run ./generate

package kbucket

import (
	"container/list"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// Bucket holds a list of peers.
type Bucket struct {
	lk   sync.RWMutex
	list *list.List
}

type PeerIDLatency struct {
	ID      peer.ID
	Latency time.Duration
}

func newBucket() *Bucket {
	b := new(Bucket)
	b.list = list.New()
	return b
}

func (b *Bucket) Peers() []peer.ID {
	b.lk.RLock()
	defer b.lk.RUnlock()
	ps := make([]peer.ID, 0, b.list.Len())
	for e := b.list.Front(); e != nil; e = e.Next() {
		id := e.Value.(PeerIDLatency).ID
		ps = append(ps, id)
	}
	return ps
}

func (b *Bucket) Has(id peer.ID) bool {
	b.lk.RLock()
	defer b.lk.RUnlock()
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerIDLatency).ID == id {
			return true
		}
	}
	return false
}

func (b *Bucket) Remove(id peer.ID) bool {
	b.lk.Lock()
	defer b.lk.Unlock()
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerIDLatency).ID == id {
			b.list.Remove(e)
			return true
		}
	}
	return false
}

func (b *Bucket) MoveToFront(id peer.ID) {
	b.lk.Lock()
	defer b.lk.Unlock()
	for e := b.list.Front(); e != nil; e = e.Next() {
		if e.Value.(PeerIDLatency).ID == id {
			b.list.MoveToFront(e)
		}
	}
}

func (b *Bucket) PushFront(p peer.ID) {
	b.lk.Lock()
	b.list.PushFront(PeerIDLatency{p, 0})
	b.lk.Unlock()
}

func (b *Bucket) PushFrontWithLatency(p peer.ID, latency time.Duration) {
	b.lk.Lock()
	defer b.lk.Unlock()

	e := b.list.Front()
	for ; e != nil; e = e.Next() {
		if e.Value.(PeerIDLatency).Latency > latency {
			break
		}
	}
	elem := PeerIDLatency{p, latency}

	//e==nil means this node has the highest latency so push it to last
	if e == nil {
		b.list.PushBack(elem)
		return
	}

	//e.Prev() == nil means this node is the first and has least latency so push it to front
	if e.Prev() == nil {
		b.list.PushFront(elem)
		return
	}

	//push the current peer just before the peer which has higher latency to it
	b.list.InsertAfter(elem, e.Prev())

}

func (b *Bucket) PopBack() peer.ID {
	b.lk.Lock()
	defer b.lk.Unlock()
	last := b.list.Back()
	b.list.Remove(last)
	return last.Value.(PeerIDLatency).ID
}

func (b *Bucket) Len() int {
	b.lk.RLock()
	defer b.lk.RUnlock()
	return b.list.Len()
}

// Split splits a buckets peers into two buckets, the methods receiver will have
// peers with CPL equal to cpl, the returned bucket will have peers with CPL
// greater than cpl (returned bucket has closer peers)
func (b *Bucket) Split(cpl int, target ID) *Bucket {
	b.lk.Lock()
	defer b.lk.Unlock()

	out := list.New()
	newbuck := newBucket()
	newbuck.list = out
	e := b.list.Front()
	for e != nil {
		peerID := ConvertPeerID(e.Value.(PeerIDLatency).ID)
		peerCPL := CommonPrefixLen(peerID, target)
		if peerCPL > cpl {
			cur := e
			out.PushBack(e.Value)
			e = e.Next()
			b.list.Remove(cur)
			continue
		}
		e = e.Next()
	}
	return newbuck
}
