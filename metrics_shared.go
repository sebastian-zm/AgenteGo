package main

import (
	"log"

	"github.com/hamba/avro/v2"
)

type CPUStats struct {
	Percent   float64 `avro:"percent"`
	User      float64 `avro:"user"`
	System    float64 `avro:"system"`
	Idle      float64 `avro:"idle"`
	Iowait    float64 `avro:"iowait"`
	Steal     float64 `avro:"steal"`
	LoadAvg1  float64 `avro:"load_avg_1m"`
	LoadAvg5  float64 `avro:"load_avg_5m"`
	LoadAvg15 float64 `avro:"load_avg_15m"`
	FreqMHz   float64 `avro:"freq_mhz"`
	Model     string  `avro:"model"`
}

type MemStats struct {
	Total       int64   `avro:"total"`
	Available   int64   `avro:"available"`
	Used        int64   `avro:"used"`
	UsedPercent float64 `avro:"used_percent"`
	Buffers     int64   `avro:"buffers"`
	Cached      int64   `avro:"cached"`
	SwapTotal   int64   `avro:"swap_total"`
	SwapUsed    int64   `avro:"swap_used"`
	SwapFree    int64   `avro:"swap_free"`
	SwapPercent float64 `avro:"swap_percent"`
	SwapIn      int64   `avro:"swap_in"`
	SwapOut     int64   `avro:"swap_out"`
}

type DiskStats struct {
	Mountpoint  string  `avro:"mountpoint"`
	Device      string  `avro:"device"`
	Fstype      string  `avro:"fstype"`
	Total       int64   `avro:"total"`
	Used        int64   `avro:"used"`
	UsedPercent float64 `avro:"used_percent"`
	InodeTotal  int64   `avro:"inode_total"`
	InodeUsed   int64   `avro:"inode_used"`
	InodeFree   int64   `avro:"inode_free"`
	ReadBytes   int64   `avro:"read_bytes"`
	WriteBytes  int64   `avro:"write_bytes"`
	ReadCount   int64   `avro:"read_count"`
	WriteCount  int64   `avro:"write_count"`
	IoTimeMs    int64   `avro:"io_time_ms"`
}

type NetStats struct {
	Interface   string `avro:"interface"`
	BytesSent   int64  `avro:"bytes_sent"`
	BytesRecv   int64  `avro:"bytes_recv"`
	PacketsSent int64  `avro:"packets_sent"`
	PacketsRecv int64  `avro:"packets_recv"`
	ErrIn       int64  `avro:"err_in"`
	ErrOut      int64  `avro:"err_out"`
	DropIn      int64  `avro:"drop_in"`
	DropOut     int64  `avro:"drop_out"`
}

type TempSensor struct {
	Name  string  `avro:"name"`
	TempC float64 `avro:"temp_c"`
}

type ProcessInfo struct {
	PID    int32   `avro:"pid"`
	Name   string  `avro:"name"`
	CPUPct float64 `avro:"cpu_pct"`
	MemRSS int64   `avro:"mem_rss"`
}

type OSStats struct {
	Hostname        string `avro:"hostname"`
	Uptime          int64  `avro:"uptime"`
	OS              string `avro:"os"`
	Platform        string `avro:"platform"`
	PlatformVersion string `avro:"platform_version"`
	KernelVersion   string `avro:"kernel_version"`
	Procs           int64  `avro:"procs"`
}

type Metrics struct {
	Timestamp   int64         `avro:"timestamp"`
	CPU         CPUStats      `avro:"cpu"`
	Memory      MemStats      `avro:"memory"`
	Disks       []DiskStats   `avro:"disks"`
	Network     []NetStats    `avro:"network"`
	Temps       []TempSensor  `avro:"temps"`
	TopCPUProcs []ProcessInfo `avro:"top_cpu_procs"`
	TopMemProcs []ProcessInfo `avro:"top_mem_procs"`
	OS          OSStats       `avro:"os"`
	ActiveConns int64         `avro:"active_connections"`
}

var metricsSchema avro.Schema

func initMetricsSchema() {
	schemaStr := `{
		"type": "record",
		"name": "Metrics",
		"namespace": "com.example",
		"fields": [
			{"name": "timestamp", "type": "long"},
			{"name": "cpu", "type": {"type": "record", "name": "CPUStats", "fields": [
				{"name": "percent",      "type": "double"},
				{"name": "user",         "type": "double"},
				{"name": "system",       "type": "double"},
				{"name": "idle",         "type": "double"},
				{"name": "iowait",       "type": "double"},
				{"name": "steal",        "type": "double"},
				{"name": "load_avg_1m",  "type": "double"},
				{"name": "load_avg_5m",  "type": "double"},
				{"name": "load_avg_15m", "type": "double"},
				{"name": "freq_mhz",     "type": "double"},
				{"name": "model",        "type": "string"}
			]}},
			{"name": "memory", "type": {"type": "record", "name": "MemStats", "fields": [
				{"name": "total",        "type": "long"},
				{"name": "available",    "type": "long"},
				{"name": "used",         "type": "long"},
				{"name": "used_percent", "type": "double"},
				{"name": "buffers",      "type": "long"},
				{"name": "cached",       "type": "long"},
				{"name": "swap_total",   "type": "long"},
				{"name": "swap_used",    "type": "long"},
				{"name": "swap_free",    "type": "long"},
				{"name": "swap_percent", "type": "double"},
				{"name": "swap_in",      "type": "long"},
				{"name": "swap_out",     "type": "long"}
			]}},
			{"name": "disks", "type": {"type": "array", "items": {"type": "record", "name": "DiskStats", "fields": [
				{"name": "mountpoint",   "type": "string"},
				{"name": "device",       "type": "string"},
				{"name": "fstype",       "type": "string"},
				{"name": "total",        "type": "long"},
				{"name": "used",         "type": "long"},
				{"name": "used_percent", "type": "double"},
				{"name": "inode_total",  "type": "long"},
				{"name": "inode_used",   "type": "long"},
				{"name": "inode_free",   "type": "long"},
				{"name": "read_bytes",   "type": "long"},
				{"name": "write_bytes",  "type": "long"},
				{"name": "read_count",   "type": "long"},
				{"name": "write_count",  "type": "long"},
				{"name": "io_time_ms",   "type": "long"}
			]}}},
			{"name": "network", "type": {"type": "array", "items": {"type": "record", "name": "NetStats", "fields": [
				{"name": "interface",    "type": "string"},
				{"name": "bytes_sent",   "type": "long"},
				{"name": "bytes_recv",   "type": "long"},
				{"name": "packets_sent", "type": "long"},
				{"name": "packets_recv", "type": "long"},
				{"name": "err_in",       "type": "long"},
				{"name": "err_out",      "type": "long"},
				{"name": "drop_in",      "type": "long"},
				{"name": "drop_out",     "type": "long"}
			]}}},
			{"name": "temps", "type": {"type": "array", "items": {"type": "record", "name": "TempSensor", "fields": [
				{"name": "name",   "type": "string"},
				{"name": "temp_c", "type": "double"}
			]}}},
			{"name": "top_cpu_procs", "type": {"type": "array", "items": {"type": "record", "name": "ProcessInfo", "fields": [
				{"name": "pid",     "type": "int"},
				{"name": "name",    "type": "string"},
				{"name": "cpu_pct", "type": "double"},
				{"name": "mem_rss", "type": "long"}
			]}}},
			{"name": "top_mem_procs", "type": {"type": "array", "items": "ProcessInfo"}},
			{"name": "os", "type": {"type": "record", "name": "OSStats", "fields": [
				{"name": "hostname",         "type": "string"},
				{"name": "uptime",           "type": "long"},
				{"name": "os",               "type": "string"},
				{"name": "platform",         "type": "string"},
				{"name": "platform_version", "type": "string"},
				{"name": "kernel_version",   "type": "string"},
				{"name": "procs",            "type": "long"}
			]}},
			{"name": "active_connections", "type": "long"}
		]
	}`
	var err error
	metricsSchema, err = avro.Parse(schemaStr)
	if err != nil {
		log.Fatalf("Error parsing avro schema: %v", err)
	}
}

func init() {
	initMetricsSchema()
}
