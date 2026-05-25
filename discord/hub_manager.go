package discord

import "hash/fnv"

type HubManager struct {
	shards []*Hub
}

func NewHubManager(n int) *HubManager {
	h := &HubManager{shards: make([]*Hub, n)}
	for i := range h.shards {
		h.shards[i] = NewHub()
		go h.shards[i].Run()
	}
	return h
}

func (m *HubManager) ForRoom(room string) *Hub {
	h := fnv.New32a()
	h.Write([]byte(room))
	return m.shards[int(h.Sum32())%len(m.shards)]
}
