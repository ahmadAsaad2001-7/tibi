package ws

import (
	"log/slog"
	"strconv"
	"sync"
)

type Hub interface {
	SendToUser(userID int64, event Event)
	SendToConsultation(consultationID int64, event Event)
	SendToQueue(clinicSessionID int64, event Event)
}

type hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	byUser  map[int64]map[*Client]struct{}
	byGroup map[string]map[*Client]struct{}
	checker MembershipChecker
	log     *slog.Logger
}

func NewHub(checker MembershipChecker, log *slog.Logger) Hub {
	if log == nil {
		log = slog.Default()
	}
	return &hub{
		clients: map[*Client]struct{}{},
		byUser:  map[int64]map[*Client]struct{}{},
		byGroup: map[string]map[*Client]struct{}{},
		checker: checker,
		log:     log,
	}
}

func NewNoop() Hub { return noopHub{} }

type noopHub struct{}

func (noopHub) SendToUser(int64, Event)         {}
func (noopHub) SendToConsultation(int64, Event) {}
func (noopHub) SendToQueue(int64, Event)        {}

func (h *hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	if h.byUser[c.userID] == nil {
		h.byUser[c.userID] = map[*Client]struct{}{}
	}
	h.byUser[c.userID][c] = struct{}{}
}

func (h *hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	if h.byUser[c.userID] != nil {
		delete(h.byUser[c.userID], c)
		if len(h.byUser[c.userID]) == 0 {
			delete(h.byUser, c.userID)
		}
	}
	for g := range c.groups {
		if h.byGroup[g] != nil {
			delete(h.byGroup[g], c)
			if len(h.byGroup[g]) == 0 {
				delete(h.byGroup, g)
			}
		}
	}
}

func (h *hub) join(c *Client, group string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byGroup[group] == nil {
		h.byGroup[group] = map[*Client]struct{}{}
	}
	h.byGroup[group][c] = struct{}{}
	c.groups[group] = struct{}{}
}

func (h *hub) leave(c *Client, group string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.byGroup[group] != nil {
		delete(h.byGroup[group], c)
		if len(h.byGroup[group]) == 0 {
			delete(h.byGroup, group)
		}
	}
	delete(c.groups, group)
}

func (h *hub) sendToGroup(group string, event Event) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.byGroup[group]))
	for c := range h.byGroup[group] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.enqueue(event)
	}
}

func (h *hub) SendToUser(userID int64, event Event) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.byUser[userID]))
	for c := range h.byUser[userID] {
		targets = append(targets, c)
	}
	h.mu.RUnlock()
	for _, c := range targets {
		c.enqueue(event)
	}
}

func (h *hub) SendToConsultation(consultationID int64, event Event) {
	h.sendToGroup(consultationGroup(consultationID), event)
}

func (h *hub) SendToQueue(clinicSessionID int64, event Event) {
	h.sendToGroup(queueGroup(clinicSessionID), event)
}

func consultationGroup(id int64) string { return "consultation_" + strconv.FormatInt(id, 10) }
func queueGroup(id int64) string        { return "queue_" + strconv.FormatInt(id, 10) }
func userGroup(id int64) string         { return "user_" + strconv.FormatInt(id, 10) }
