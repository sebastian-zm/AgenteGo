package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hamba/avro/v2"
	"github.com/rabbitmq/amqp091-go"
)

func printMetrics(m Metrics) {
	ts := time.Unix(m.Timestamp, 0).Format("2006-01-02 15:04:05")
	fmt.Printf("\n✅ Métricas recibidas [%s]\n", ts)
	fmt.Printf("  CPU:   %.1f%% | user=%.1f%% sys=%.1f%% iowait=%.1f%% idle=%.1f%% | load=%.2f/%.2f/%.2f | %.0fMHz | %s\n",
		m.CPU.Percent, m.CPU.User, m.CPU.System, m.CPU.Iowait, m.CPU.Idle,
		m.CPU.LoadAvg1, m.CPU.LoadAvg5, m.CPU.LoadAvg15,
		m.CPU.FreqMHz, m.CPU.Model)
	fmt.Printf("  RAM:   %.1f%% | used=%d MB avail=%d MB total=%d MB | buffers=%d MB cached=%d MB\n",
		m.Memory.UsedPercent,
		m.Memory.Used/1024/1024, m.Memory.Available/1024/1024, m.Memory.Total/1024/1024,
		m.Memory.Buffers/1024/1024, m.Memory.Cached/1024/1024)
	fmt.Printf("  Swap:  %.1f%% | used=%d MB free=%d MB | in=%d out=%d\n",
		m.Memory.SwapPercent,
		m.Memory.SwapUsed/1024/1024, m.Memory.SwapFree/1024/1024,
		m.Memory.SwapIn, m.Memory.SwapOut)
	for _, d := range m.Disks {
		fmt.Printf("  Disk:  %s (%s) %.1f%% | %d GB / %d GB | inodes=%d/%d | r=%d MB w=%d MB\n",
			d.Mountpoint, d.Fstype, d.UsedPercent,
			d.Used/1024/1024/1024, d.Total/1024/1024/1024,
			d.InodeUsed, d.InodeTotal,
			d.ReadBytes/1024/1024, d.WriteBytes/1024/1024)
	}
	for _, n := range m.Network {
		fmt.Printf("  Net:   %s | tx=%d KB rx=%d KB | err_in=%d err_out=%d drop_in=%d drop_out=%d\n",
			n.Interface,
			n.BytesSent/1024, n.BytesRecv/1024,
			n.ErrIn, n.ErrOut, n.DropIn, n.DropOut)
	}
	if len(m.Temps) > 0 {
		fmt.Printf("  Temp:  ")
		for _, t := range m.Temps {
			fmt.Printf("%s=%.1f°C  ", t.Name, t.TempC)
		}
		fmt.Println()
	}
	fmt.Printf("  OS:    %s %s | kernel=%s | uptime=%dd %dh | procs=%d | conns=%d\n",
		m.OS.Platform, m.OS.PlatformVersion, m.OS.KernelVersion,
		m.OS.Uptime/86400, (m.OS.Uptime%86400)/3600,
		m.OS.Procs, m.ActiveConns)
	if len(m.TopCPUProcs) > 0 {
		fmt.Printf("  Top CPU: ")
		for _, p := range m.TopCPUProcs {
			fmt.Printf("%s(%.1f%%) ", p.Name, p.CPUPct)
		}
		fmt.Println()
	}
	if len(m.TopMemProcs) > 0 {
		fmt.Printf("  Top MEM: ")
		for _, p := range m.TopMemProcs {
			fmt.Printf("%s(%d MB) ", p.Name, p.MemRSS/1024/1024)
		}
		fmt.Println()
	}
}

func runConsumer() {
	rabbitMQURL := flag.String("rabbitmq-url", "amqp://guest:guest@localhost:5672/", "RabbitMQ connection URL")
	flag.Parse()

	conn, err := amqp091.Dial(*rabbitMQURL)
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error abriendo canal: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("it_metrics", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error declarando cola: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Error registrando el consumidor: %v", err)
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			var payload Metrics
			if err := avro.Unmarshal(metricsSchema, d.Body, &payload); err != nil {
				log.Printf("Error deserializando Avro: %v", err)
				continue
			}
			printMetrics(payload)
		}
	}()

	log.Printf(" [*] Escuchando mensajes en la cola '%s'. Para salir pulsa CTRL+C", q.Name)
	<-forever
}
