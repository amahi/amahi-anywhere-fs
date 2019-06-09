package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"strconv"
	"strings"
	"time"
)

var DB *gorm.DB

type SystemStatusModel struct {
	gorm.Model
	Total          uint64
	Free           uint64
	RAMUsedPercent float64
	CPUUsedPercent string
}

func CreateSSModelObject(status SystemStatus) (SystemStatusModel) {
	str := ""
	for _, v := range status.CPUUsedPercent {
		if str == "" {
			str = fmt.Sprintf("%v", v)
		}
		str = fmt.Sprintf("%s|%v", str, v)
	}

	ss := SystemStatusModel{
		Total:          status.Total,
		Free:           status.Free,
		RAMUsedPercent: status.RAMUsedPercent,
		CPUUsedPercent: str,
	}
	return ss
}

type SystemStatus struct {
	// Memory
	Total          uint64  `json:"ram_total"`
	Free           uint64  `json:"ram_free"`
	RAMUsedPercent float64 `json:"ram_used_percent"`

	// CPU
	CPUUsedPercent []float64 `json:"cpu_used_percent"`
}

func GetSystemStatus() (SystemStatus, error) {
	// 3 second halt to get average CPU usage
	c, err := cpu.Percent(time.Duration(3000000000), true)

	if err != nil {
		return SystemStatus{}, err
	}

	r, err := mem.VirtualMemory()
	if err != nil {
		return SystemStatus{}, err
	}

	ss := SystemStatus{
		Total:          r.Total,
		Free:           r.Free,
		RAMUsedPercent: r.UsedPercent,
		CPUUsedPercent: c,
	}
	return ss, nil
}

func GetSystemStatusFromModel(model SystemStatusModel) SystemStatus {
	var p []float64
	str := strings.Split(model.CPUUsedPercent, "|")
	for _, v := range str {
		f, _ := strconv.ParseFloat(v, 64)
		p = append(p, f)
	}
	ss := SystemStatus{
		Total:          model.Total,
		Free:           model.Free,
		RAMUsedPercent: model.RAMUsedPercent,
		CPUUsedPercent: p,
	}
	return ss
}

func GetSystemStatusFromModelList(models []SystemStatusModel) []SystemStatus {
	var ss []SystemStatus
	for _, model := range models {
		var p []float64
		str := strings.Split(model.CPUUsedPercent, "|")
		for _, v := range str {
			f, _ := strconv.ParseFloat(v, 64)
			p = append(p, f)
		}
		s := SystemStatus{
			Total:          model.Total,
			Free:           model.Free,
			RAMUsedPercent: model.RAMUsedPercent,
			CPUUsedPercent: p,
		}
		ss = append(ss, s)
	}
	return ss
}

func SystemStatusRoutine() {
	// TODO: import the
	DB, err := gorm.Open("mysql", "vishwas:vpass@tcp(172.17.0.2)/amahi?parseTime=true")
	if err != nil {
		fmt.Println(err)
		panic("failed to connect database")
		return
	}
	defer DB.Close()
	DB.AutoMigrate(&SystemStatusModel{})

	//var ssm []SystemStatusModel
	//DB.Order("id desc").Limit(3).Find(&ssm)
	//ss := GetSystemStatusFromModelList(ssm)
	//fmt.Println(ss)
	//json.NewEncoder(os.Stdout).Encode(ss)

	for {
		// take reading after every 15 second
		time.Sleep(15 * time.Second)
		// this call will take about 3 seconds to complete
		status, _ := GetSystemStatus()
		fmt.Println("status: ", status)
		s := CreateSSModelObject(status)
		fmt.Println("its here", s)
		DB.Create(&s)
		fmt.Println("saved to db")
	}
}
