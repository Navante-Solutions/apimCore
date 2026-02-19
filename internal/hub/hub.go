package hub

import (
	"time"
)

type EventType int

const (
	TraceEvent EventType = iota
	StatUpdateEvent
)

type SystemStats struct {
	TotalRequests   int64
	AvgLatency      float64
	RateLimited     int64
	Blocked         int64
	ActiveConns     int
	Uptime          time.Duration
	CPUUsage        float64
	MemoryUsageMB   uint64
	MemoryTotalMB   uint64
	GeoThreats      map[string]int
}

type TrafficEvent struct {
	Timestamp time.Time
	Method    string
	Path      string
	Backend   string
	Status    int
	Latency   int64
	TenantID  string
	Country   string
	IP        string
	Action    string // "ALLOWED", "BLOCKED", "RATE_LIMIT"
}

// Broadcaster manages telemetry distribution
type Broadcaster struct {
	trafficChan chan TrafficEvent
	statsChan   chan SystemStats
	stopChan    chan struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		trafficChan: make(chan TrafficEvent, 100), // Buffered to handle TUI lag
		statsChan:   make(chan SystemStats, 10),
		stopChan:    make(chan struct{}),
	}
}

// PublishTraffic sends an event to the TUI (non-blocking)
func (b *Broadcaster) PublishTraffic(ev TrafficEvent) {
	select {
	case b.trafficChan <- ev:
	default:
		// Drop event if TUI is saturated to preserve Gateway performance
	}
}

// PublishStats sends periodic metrics to the TUI (non-blocking)
func (b *Broadcaster) PublishStats(s SystemStats) {
	select {
	case b.statsChan <- s:
	default:
	}
}

func (b *Broadcaster) TrafficChan() <-chan TrafficEvent {
	return b.trafficChan
}

func (b *Broadcaster) StatsChan() <-chan SystemStats {
	return b.statsChan
}

func (b *Broadcaster) Stop() {
	close(b.stopChan)
}

// Collector gathers system vitals
type Collector struct {
	Hub      *Broadcaster
	interval time.Duration
	stop     chan struct{}
}

func NewCollector(h *Broadcaster, interval time.Duration) *Collector {
	return &Collector{
		Hub:      h,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (c *Collector) Start() {
	ticker := time.NewTicker(c.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Gather real stats here in a real app (using gopsutil or /proc)
				// For now, we emit enriched mock data to power the premium UI
				c.Hub.PublishStats(SystemStats{
					Uptime:        time.Hour * 1, // mock
					CPUUsage:      0.45,          // mock
					MemoryUsageMB: 1240,          // mock
					ActiveConns:   42,            // mock
					GeoThreats:    map[string]int{"CN": 5, "RU": 2, "US": 1},
				})
			case <-c.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *Collector) Stop() {
	close(c.stop)
}
