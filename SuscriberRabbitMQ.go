package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hamba/avro/v2"
	"github.com/rabbitmq/amqp091-go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gopsnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

func collectCPU() CPUStats {
	t1, err := cpu.Times(false)
	if err != nil || len(t1) == 0 {
		time.Sleep(time.Second)
		return CPUStats{Model: "Unknown"}
	}

	time.Sleep(time.Second)

	t2, err := cpu.Times(false)
	stats := CPUStats{}

	if err == nil && len(t2) > 0 {
		user    := t2[0].User    - t1[0].User
		system  := t2[0].System  - t1[0].System
		idle    := t2[0].Idle    - t1[0].Idle
		iowait  := t2[0].Iowait  - t1[0].Iowait
		steal   := t2[0].Steal   - t1[0].Steal
		nice    := t2[0].Nice    - t1[0].Nice
		irq     := t2[0].Irq     - t1[0].Irq
		softirq := t2[0].Softirq - t1[0].Softirq
		total   := user + system + idle + iowait + steal + nice + irq + softirq
		if total > 0 {
			stats.User    = user    / total * 100
			stats.System  = system  / total * 100
			stats.Idle    = idle    / total * 100
			stats.Iowait  = iowait  / total * 100
			stats.Steal   = steal   / total * 100
			stats.Percent = 100 - stats.Idle
		}
	}

	if avg, err := load.Avg(); err == nil {
		stats.LoadAvg1  = avg.Load1
		stats.LoadAvg5  = avg.Load5
		stats.LoadAvg15 = avg.Load15
	}

	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		stats.Model   = info[0].ModelName
		stats.FreqMHz = info[0].Mhz
	} else {
		stats.Model = "Unknown"
	}

	return stats
}

func collectMemory() MemStats {
	v, err := mem.VirtualMemory()
	if err != nil {
		return MemStats{}
	}

	ms := MemStats{
		Total:       int64(v.Total),
		Available:   int64(v.Available),
		Used:        int64(v.Used),
		UsedPercent: v.UsedPercent,
		Buffers:     int64(v.Buffers),
		Cached:      int64(v.Cached),
	}

	if s, err := mem.SwapMemory(); err == nil {
		ms.SwapTotal   = int64(s.Total)
		ms.SwapUsed    = int64(s.Used)
		ms.SwapFree    = int64(s.Free)
		ms.SwapPercent = s.UsedPercent
		ms.SwapIn      = int64(s.Sin)
		ms.SwapOut     = int64(s.Sout)
	}

	return ms
}

func collectDisks() []DiskStats {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return []DiskStats{}
	}

	ioCounters, _ := disk.IOCounters()

	var result []DiskStats
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		ds := DiskStats{
			Mountpoint:  p.Mountpoint,
			Device:      p.Device,
			Fstype:      p.Fstype,
			Total:       int64(usage.Total),
			Used:        int64(usage.Used),
			UsedPercent: usage.UsedPercent,
			InodeTotal:  int64(usage.InodesTotal),
			InodeUsed:   int64(usage.InodesUsed),
			InodeFree:   int64(usage.InodesFree),
		}

		if ioCounters != nil {
			if io, ok := ioCounters[filepath.Base(p.Device)]; ok {
				ds.ReadBytes  = int64(io.ReadBytes)
				ds.WriteBytes = int64(io.WriteBytes)
				ds.ReadCount  = int64(io.ReadCount)
				ds.WriteCount = int64(io.WriteCount)
				ds.IoTimeMs   = int64(io.IoTime)
			}
		}

		result = append(result, ds)
	}

	if result == nil {
		return []DiskStats{}
	}
	return result
}

func collectNetwork() ([]NetStats, int64) {
	counters, err := gopsnet.IOCounters(true)
	var netStats []NetStats
	if err == nil {
		for _, c := range counters {
			netStats = append(netStats, NetStats{
				Interface:   c.Name,
				BytesSent:   int64(c.BytesSent),
				BytesRecv:   int64(c.BytesRecv),
				PacketsSent: int64(c.PacketsSent),
				PacketsRecv: int64(c.PacketsRecv),
				ErrIn:       int64(c.Errin),
				ErrOut:      int64(c.Errout),
				DropIn:      int64(c.Dropin),
				DropOut:     int64(c.Dropout),
			})
		}
	}
	if netStats == nil {
		netStats = []NetStats{}
	}

	var activeConns int64
	if conns, err := gopsnet.Connections("all"); err == nil {
		activeConns = int64(len(conns))
	}

	return netStats, activeConns
}

func collectTemps() []TempSensor {
	sensors, err := host.SensorsTemperatures()
	if err == nil && len(sensors) > 0 {
		result := make([]TempSensor, 0, len(sensors))
		for _, s := range sensors {
			if s.Temperature > 0 {
				result = append(result, TempSensor{Name: s.SensorKey, TempC: s.Temperature})
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	if runtime.GOOS == "windows" {
		if t := getTemperatureViaPowerShell(); t != nil {
			return []TempSensor{{Name: "cpu_thermal", TempC: *t}}
		}
	}

	return []TempSensor{}
}

func getTemperatureViaPowerShell() *float64 {
	psCmd := `Get-WmiObject -Namespace "root\wmi" -Class MSAcpi_ThermalZoneTemperature -ErrorAction SilentlyContinue | Select-Object -First 1 | ForEach-Object { $_.CurrentTemperature }`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	output, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		if val, parseErr := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); parseErr == nil {
			tempC := val/10.0 - 273.15
			if tempC > -50 && tempC < 150 {
				return &tempC
			}
		}
	}

	psCmd = `Get-WmiObject Win32_SystemEnclosure -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty HotSwappable`
	cmd = exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	output, err = cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		if val, parseErr := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); parseErr == nil && val > -50 && val < 150 {
			return &val
		}
	}

	psCmd = `Get-WmiObject Win32_TemperatureProbe -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty CurrentReading`
	cmd = exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	output, err = cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		if val, parseErr := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); parseErr == nil {
			if val > 1000 {
				val = val / 10.0
			}
			if val > -50 && val < 150 {
				return &val
			}
		}
	}

	return nil
}

func collectProcesses(n int) (byCPU, byMem []ProcessInfo) {
	procs, err := process.Processes()
	if err != nil {
		return []ProcessInfo{}, []ProcessInfo{}
	}

	type entry struct {
		pid    int32
		name   string
		cpu    float64
		memRSS int64
	}

	entries := make([]entry, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		var rss int64
		if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
			rss = int64(memInfo.RSS)
		}
		entries = append(entries, entry{p.Pid, name, cpuPct, rss})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].cpu > entries[j].cpu })
	for i := 0; i < n && i < len(entries); i++ {
		e := entries[i]
		byCPU = append(byCPU, ProcessInfo{e.pid, e.name, e.cpu, e.memRSS})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].memRSS > entries[j].memRSS })
	for i := 0; i < n && i < len(entries); i++ {
		e := entries[i]
		byMem = append(byMem, ProcessInfo{e.pid, e.name, e.cpu, e.memRSS})
	}

	if byCPU == nil {
		byCPU = []ProcessInfo{}
	}
	if byMem == nil {
		byMem = []ProcessInfo{}
	}
	return
}

func collectOS() OSStats {
	info, err := host.Info()
	if err != nil {
		return OSStats{}
	}
	return OSStats{
		Hostname:        info.Hostname,
		Uptime:          int64(info.Uptime),
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		Procs:           int64(info.Procs),
	}
}

func sendMetrics(rabbitURL string) {
	cpuStats              := collectCPU() // bloquea 1 segundo midiendo tiempos
	memStats              := collectMemory()
	diskStats             := collectDisks()
	netStats, activeConns := collectNetwork()
	temps                 := collectTemps()
	topCPU, topMem       := collectProcesses(10)
	osStats               := collectOS()

	payload := Metrics{
		Timestamp:   time.Now().Unix(),
		CPU:         cpuStats,
		Memory:      memStats,
		Disks:       diskStats,
		Network:     netStats,
		Temps:       temps,
		TopCPUProcs: topCPU,
		TopMemProcs: topMem,
		OS:          osStats,
		ActiveConns: activeConns,
	}

	fmt.Printf("[%s] CPU: %.1f%% (user=%.1f%% sys=%.1f%% idle=%.1f%%) load=[%.2f %.2f %.2f] | RAM: %.1f%% | discos: %d | interfaces: %d | procs: %d\n",
		time.Now().Format("2006-01-02 15:04:05"),
		payload.CPU.Percent, payload.CPU.User, payload.CPU.System, payload.CPU.Idle,
		payload.CPU.LoadAvg1, payload.CPU.LoadAvg5, payload.CPU.LoadAvg15,
		payload.Memory.UsedPercent,
		len(payload.Disks), len(payload.Network), osStats.Procs,
	)

	body, err := avro.Marshal(metricsSchema, payload)
	if err != nil {
		log.Printf("Error al serializar a Avro: %v", err)
		return
	}

	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		log.Printf("Error conectando a RabbitMQ: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Error abriendo canal: %v", err)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("it_metrics", false, false, false, false, nil)
	if err != nil {
		log.Printf("Error declarando cola: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, "", q.Name, false, false, amqp091.Publishing{
		ContentType: "application/avro",
		Body:        body,
	})
	if err != nil {
		log.Printf("Error al enviar mensaje a RabbitMQ: %v", err)
	} else {
		fmt.Printf("[%s] Métricas enviadas correctamente (%d bytes)\n",
			time.Now().Format("2006-01-02 15:04:05"), len(body))
	}
}

func runPublisher() {
	rabbitMQURL := flag.String("rabbitmq-url", "amqp://guest:guest@localhost:5672/", "RabbitMQ connection URL")
	flag.Parse()

	fmt.Println("Iniciando servicio de recolección métricas en segundo plano...")
	fmt.Println("Se enviarán métricas a RabbitMQ cada 5 minutos.")
	fmt.Println("Para detener el servicio, mata o finaliza el proceso desde el 'Administrador de Tareas'.")

	sendMetrics(*rabbitMQURL)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sendMetrics(*rabbitMQURL)
	}
}

func main() {
	runPublisher()
}
