package main

import (
	"encoding/json"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"time"
)

type SystemStatus struct {
	// RAM
	Total          uint64  `json:"ram_total"`
	Free           uint64  `json:"ram_free"`
	RAMUsedPercent float64 `json:"ram_used_percent"`

	// CPU
	CPUUsedPercent []float64 `json:"cpu_used_percent"`
}

func GetSystemStatus() (*SystemStatus, error) {
	// 3 second halt to get average CPU usage
	c, err := cpu.Percent(time.Duration(3000000000), true)

	if err != nil {
		return nil, err
	}

	r, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	ss := SystemStatus{
		Total:          r.Total,
		Free:           r.Free,
		RAMUsedPercent: r.UsedPercent,
		CPUUsedPercent: c,
	}

	j, _ := json.Marshal(ss)
	fmt.Println(string(j))

	return &ss, nil
}

func SystemStatusRoutine() {
	for {
		// take reading after every 15 second
		time.Sleep(15 * time.Second)
		// this call will take about 3 seconds to complete
		// TODO: store the system status in database to maintain the system health history
		GetSystemStatus()
	}
}
