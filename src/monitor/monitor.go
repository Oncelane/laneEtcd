package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessStats 存储进程统计信息
type ProcessStats struct {
	PID             int32
	CPUPercent      float64
	MemoryUsed      uint64
	RunningTime     time.Duration
	RestartCount    int
	ThreadCount     int32
	StartTime       time.Time
	LastRestartTime time.Time
}

// SystemStats 存储系统统计信息
type SystemStats struct {
	CPUUsage      float64
	MemoryTotal   uint64
	MemoryUsed    uint64
	MemoryPercent float64
	DiskTotal     uint64
	DiskUsed      uint64
	DiskPercent   float64
	NetSentBytes  uint64
	NetRecvBytes  uint64
}

type RaftStates struct {
	State            string // leader/follower/cadidate
	AssumeGroupState uint8
	HeartbeatLagecy  time.Time
	LogLagecy        uint64
}

type KVState struct {
	TotalNum      uint64
	HotkeysInfo   HotKeyAnalyseInfo
	OpAnalyseInfo OpAnalyseInfo
	LockInfo      LockInfo
}

type LockInfo struct {
	// 锁数量
	// 具体锁的值
	// 锁的时间段
	// 锁竞争热点
}

type HotKeyAnalyseInfo struct {
	HotKeys []struct {
		Frquency uint64
		Key      string
	}
}

type OpAnalyseInfo struct {
}

var (
	processStats ProcessStats
	systemStats  SystemStats
	startTime    = time.Now()
	restartCount = 0
	lastNetIO    *net.IOCountersStat
)

func init() {
	// 初始化进程统计信息
	processStats = ProcessStats{
		PID:          int32(os.Getpid()),
		StartTime:    startTime,
		RestartCount: restartCount,
		RunningTime:  0,
	}

	// 获取初始网络IO数据作为基准
	if netIO, err := getNetworkIO(); err == nil && len(netIO) > 0 {
		lastNetIO = &netIO[0]
	}
}

func main() {
	// 每隔1秒更新一次监控数据
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			processStats.Update()
			systemStats.Update()
		}
	}
}

type Monitor struct {
	State *State
	conf  MonitorConf
}

type MonitorConf struct {
	BaseUrl string
}

type State struct {
	S SystemStats
	P ProcessStats
}

func (s *State) Update() {
	s.S.Update()
	s.P.Update()
}

func (m *Monitor) LoadRouter(r *gin.Engine) {
	g := r.Group("monitor")
	g.GET(m.conf.BaseUrl+"/state", m.StateGet)
}

func (m *Monitor) StateGet(c *gin.Context) {
	m.State.Update()
	c.JSON(200, m.State)
}

func (m *Monitor) LogGet(c *gin.Context) {

}

func (p *ProcessStats) Update() {
	pp, err := process.NewProcess(processStats.PID)
	if err != nil {
		log.Printf("获取进程信息失败: %v", err)
		return
	}

	// 更新CPU使用率
	if cpuPercent, err := pp.CPUPercent(); err == nil {
		p.CPUPercent = cpuPercent
	}

	// 更新内存使用
	if memInfo, err := pp.MemoryInfo(); err == nil && memInfo != nil {
		p.MemoryUsed = memInfo.RSS
	}

	// 更新线程数
	if numThreads, err := pp.NumThreads(); err == nil {
		p.ThreadCount = numThreads
	}

	// 更新运行时间
	p.RunningTime = time.Since(p.StartTime)
}

func (s *SystemStats) Update() {
	// 更新CPU使用率
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		systemStats.CPUUsage = cpuPercent[0]
	}

	// 更新内存使用情况
	if memInfo, err := mem.VirtualMemory(); err == nil {
		systemStats.MemoryTotal = memInfo.Total
		systemStats.MemoryUsed = memInfo.Used
		systemStats.MemoryPercent = memInfo.UsedPercent
	}

	// 更新磁盘使用情况
	if diskInfo, err := disk.Usage(filepath.Dir(os.Args[0])); err == nil {
		systemStats.DiskTotal = diskInfo.Total
		systemStats.DiskUsed = diskInfo.Used
		systemStats.DiskPercent = diskInfo.UsedPercent
	}

	// 更新网络IO
	if netIO, err := getNetworkIO(); err == nil && len(netIO) > 0 && lastNetIO != nil {
		systemStats.NetSentBytes = netIO[0].BytesSent - lastNetIO.BytesSent
		systemStats.NetRecvBytes = netIO[0].BytesRecv - lastNetIO.BytesRecv
		lastNetIO = &netIO[0]
	}
}

func getNetworkIO() ([]net.IOCountersStat, error) {
	return net.IOCounters(false)
}

func displayStats() {
	fmt.Println("===================================")
	fmt.Println("资源监控统计 - " + time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("===================================")

	fmt.Println("\n[进程信息]")
	fmt.Printf("PID: %d\n", processStats.PID)
	fmt.Printf("CPU使用率: %.2f%%\n", processStats.CPUPercent)
	fmt.Printf("内存使用: %.2f MB\n", float64(processStats.MemoryUsed)/1024/1024)
	fmt.Printf("线程数: %d\n", processStats.ThreadCount)
	fmt.Printf("运行时间: %s\n", processStats.RunningTime.Round(time.Second))
	fmt.Printf("重启次数: %d\n", processStats.RestartCount)
	fmt.Printf("启动时间: %s\n", processStats.StartTime.Format("2006-01-02 15:04:05"))

	fmt.Println("\n[系统信息]")
	fmt.Printf("CPU使用率: %.2f%%\n", systemStats.CPUUsage)
	fmt.Printf("内存总量: %.2f GB\n", float64(systemStats.MemoryTotal)/1024/1024/1024)
	fmt.Printf("内存使用: %.2f GB (%.2f%%)\n",
		float64(systemStats.MemoryUsed)/1024/1024/1024,
		systemStats.MemoryPercent)
	fmt.Printf("磁盘总量: %.2f GB\n", float64(systemStats.DiskTotal)/1024/1024/1024)
	fmt.Printf("磁盘使用: %.2f GB (%.2f%%)\n",
		float64(systemStats.DiskUsed)/1024/1024/1024,
		systemStats.DiskPercent)
	fmt.Printf("网络发送: %.2f KB/s\n", float64(systemStats.NetSentBytes)/1024/5)
	fmt.Printf("网络接收: %.2f KB/s\n", float64(systemStats.NetRecvBytes)/1024/5)

	fmt.Println("\n[运行环境]")
	fmt.Printf("GOOS: %s, GOARCH: %s, CPU核心: %d\n",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())

	fmt.Println("===================================")
}
