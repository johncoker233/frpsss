// Copyright 2019 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mem

import (
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/metric"
	server "github.com/fatedier/frp/server/metrics"
)
// 定义 Collector 接口
type Collector interface {
    // 统计信息收集方法
    CollectStatistics() error
}

// 定义 ServerStatistics 结构体
type ServerStatistics struct {
    // 服务器统计信息字段
    ReserveDays      int    // 保留天数
    TotalClientCount int    // 总客户端数
    OnlineCount      int    // 在线客户端数
    // 其他服务器统计相关字段
}

// 定义 ProxyStatistics 结构体 
type ProxyStatistics struct {
    // 代理统计信息字段
    Name          string    // 代理名称
    Type          string    // 代理类型
    Status        string    // 代理状态
    TodayTraffic  int64    // 今日流量
    TotalTraffic  int64    // 总流量
    // 其他代理统计相关字段
}

// 定义 ServerStats 结构体
type ServerStats struct {
    // 服务器状态统计
    UpTime       int64     // 运行时间
    ConnCount    int       // 连接数
    CPUPercent   float64   // CPU使用率
    MemUsedBytes int64     // 内存使用
    // 其他状态统计字段
}

// 定义 ProxyStats 结构体
type ProxyStats struct {
    // 代理状态统计
    Name       string     // 代理名称 
    Type       string     // 代理类型
    Traffic    int64      // 流量统计
    Bandwidth  int64      // 带宽使用
    // 其他代理状态字段
}

// 定义 ProxyTrafficInfo 结构体
type ProxyTrafficInfo struct {
    // 代理流量信息
    Name          string    // 代理名称
    TrafficIn     int64     // 入站流量
    TrafficOut    int64     // 出站流量
    CurConnCount  int       // 当前连接数
    // 其他流量相关字段
}
var (
	sm = newServerMetrics()

	ServerMetrics  server.ServerMetrics
	StatsCollector Collector
)

func init() {
	ServerMetrics = sm
	StatsCollector = sm
	sm.run()
}

type serverMetrics struct {
	info *ServerStatistics
	mu   sync.Mutex
}

func newServerMetrics() *serverMetrics {
	return &serverMetrics{
		info: &ServerStatistics{
			TotalTrafficIn:  metric.NewDateCounter(ReserveDays),
			TotalTrafficOut: metric.NewDateCounter(ReserveDays),
			CurConns:        metric.NewCounter(),

			ClientCounts:    metric.NewCounter(),
			ProxyTypeCounts: make(map[string]metric.Counter),

			ProxyStatistics: make(map[string]*ProxyStatistics),
		},
	}
}


func (m *serverMetrics) run() {
    go func() {
        for {
            time.Sleep(5 * time.Second) // 修改更新频率为每 5 秒
            start := time.Now()
            count, total := m.clearUselessInfo(time.Duration(7*24) * time.Hour)
            log.Debugf("clear useless proxy statistics data count %d/%d, cost %v", count, total, time.Since(start))
        }
    }()
}

func (m *serverMetrics) clearUselessInfo(continuousOfflineDuration time.Duration) (int, int) {
	count := 0
	total := 0
	// To check if there are any proxies that have been closed for more than continuousOfflineDuration and remove them.
	m.mu.Lock()
	defer m.mu.Unlock()
	total = len(m.info.ProxyStatistics)
	for name, data := range m.info.ProxyStatistics {
		if !data.LastCloseTime.IsZero() &&
			data.LastStartTime.Before(data.LastCloseTime) &&
			time.Since(data.LastCloseTime) > continuousOfflineDuration {
			delete(m.info.ProxyStatistics, name)
			count++
			log.Tracef("clear proxy [%s]'s statistics data, lastCloseTime: [%s]", name, data.LastCloseTime.String())
		}
	}
	return count, total
}

func (m *serverMetrics) ClearOfflineProxies() (int, int) {
	return m.clearUselessInfo(0)
}

func (m *serverMetrics) NewClient() {
	m.info.ClientCounts.Inc(1)
}

func (m *serverMetrics) CloseClient() {
	m.info.ClientCounts.Dec(1)
}

func (m *serverMetrics) NewProxy(name string, proxyType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counter, ok := m.info.ProxyTypeCounts[proxyType]
	if !ok {
		counter = metric.NewCounter()
	}
	counter.Inc(1)
	m.info.ProxyTypeCounts[proxyType] = counter

	proxyStats, ok := m.info.ProxyStatistics[name]
	if !(ok && proxyStats.ProxyType == proxyType) {
		proxyStats = &ProxyStatistics{
			Name:       name,
			ProxyType:  proxyType,
			CurConns:   metric.NewCounter(),
			TrafficIn:  metric.NewDateCounter(ReserveDays),
			TrafficOut: metric.NewDateCounter(ReserveDays),
		}
		m.info.ProxyStatistics[name] = proxyStats
	}
	proxyStats.LastStartTime = time.Now()
}

func (m *serverMetrics) CloseProxy(name string, proxyType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if counter, ok := m.info.ProxyTypeCounts[proxyType]; ok {
		counter.Dec(1)
	}
	if proxyStats, ok := m.info.ProxyStatistics[name]; ok {
		proxyStats.LastCloseTime = time.Now()
	}
}

func (m *serverMetrics) OpenConnection(name string, _ string) {
	m.info.CurConns.Inc(1)

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.CurConns.Inc(1)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

func (m *serverMetrics) CloseConnection(name string, _ string) {
	m.info.CurConns.Dec(1)

	m.mu.Lock()
	defer m.mu.Unlock()
	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.CurConns.Dec(1)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

func (m *serverMetrics) AddTrafficIn(name string, _ string, trafficBytes int64) {
	m.info.TotalTrafficIn.Inc(trafficBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.TrafficIn.Inc(trafficBytes)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

func (m *serverMetrics) AddTrafficOut(name string, _ string, trafficBytes int64) {
	m.info.TotalTrafficOut.Inc(trafficBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		proxyStats.TrafficOut.Inc(trafficBytes)
		m.info.ProxyStatistics[name] = proxyStats
	}
}

// Get stats data api.

func (m *serverMetrics) GetServer() *ServerStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &ServerStats{
		TotalTrafficIn:  m.info.TotalTrafficIn.TodayCount(),
		TotalTrafficOut: m.info.TotalTrafficOut.TodayCount(),
		CurConns:        int64(m.info.CurConns.Count()),
		ClientCounts:    int64(m.info.ClientCounts.Count()),
		ProxyTypeCounts: make(map[string]int64),
	}
	for k, v := range m.info.ProxyTypeCounts {
		s.ProxyTypeCounts[k] = int64(v.Count())
	}
	return s
}

func (m *serverMetrics) GetProxiesByType(proxyType string) []*ProxyStats {
	res := make([]*ProxyStats, 0)
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, proxyStats := range m.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		ps := &ProxyStats{
			Name:            name,
			Type:            proxyStats.ProxyType,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        int64(proxyStats.CurConns.Count()),
		}
		if !proxyStats.LastStartTime.IsZero() {
			ps.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			ps.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		res = append(res, ps)
	}
	return res
}

func (m *serverMetrics) GetProxiesByTypeAndName(proxyType string, proxyName string) (res *ProxyStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, proxyStats := range m.info.ProxyStatistics {
		if proxyStats.ProxyType != proxyType {
			continue
		}

		if name != proxyName {
			continue
		}

		res = &ProxyStats{
			Name:            name,
			Type:            proxyStats.ProxyType,
			TodayTrafficIn:  proxyStats.TrafficIn.TodayCount(),
			TodayTrafficOut: proxyStats.TrafficOut.TodayCount(),
			CurConns:        int64(proxyStats.CurConns.Count()),
		}
		if !proxyStats.LastStartTime.IsZero() {
			res.LastStartTime = proxyStats.LastStartTime.Format("01-02 15:04:05")
		}
		if !proxyStats.LastCloseTime.IsZero() {
			res.LastCloseTime = proxyStats.LastCloseTime.Format("01-02 15:04:05")
		}
		break
	}
	return
}

func (m *serverMetrics) GetProxyTraffic(name string) (res *ProxyTrafficInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxyStats, ok := m.info.ProxyStatistics[name]
	if ok {
		res = &ProxyTrafficInfo{
			Name: name,
		}
		res.TrafficIn = proxyStats.TrafficIn.GetLastDaysCount(ReserveDays)
		res.TrafficOut = proxyStats.TrafficOut.GetLastDaysCount(ReserveDays)
	}
	return
}