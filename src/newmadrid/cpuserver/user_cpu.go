package main

import (
	"log"
	"newmadrid/msp43x"
	"bytes"
	"bufio"
	"errors"
)

type CpuReq struct {
	Name  string
	Reply chan *UserCpu
}

var getCpu = make(chan CpuReq)

func GetCpu(name string) *UserCpu {
	reply := make(chan *UserCpu)

	getCpu <- CpuReq{
		Name:  name,
		Reply: reply,
	}

	return <-reply
}

func CpuController(redis *RedisLand) {
	cpus := make(map[string]*UserCpu)

	for {
		req := <-getCpu

		var (
			cpu *UserCpu
			ok  bool
		)

		if cpu, ok = cpus[req.Name]; !ok {
			cpu = NewUserCpu()
			cpu.Name = req.Name
			cpu.redis = redis
			cpus[req.Name] = cpu
			go cpu.Loop()
		}

		log.Printf("%p\n", cpu.MCU)

		req.Reply <- cpu
	}
}

func newMemory() *msp43x.HookableMemory {
	m := new(msp43x.HookableMemory)

	return m
}

const (
	CpuStopped = iota
	CpuRunning
	CpuFault
	CpuStepping
)

const (
	BreakStop = iota
	BreakStep 
	BreakTrace
)

type CpuRequest func(c *UserCpu) 

type UserCpu struct {
	MCU *msp43x.CPU
	Mem *msp43x.HookableMemory

	Name string

	Image string

	State int

	redis *RedisLand

	Breakpoints map[uint16]int	

	Comm chan CpuRequest

}

func NewUserCpu() (ret *UserCpu) {
	ret = new(UserCpu)
	ret.MCU = new(msp43x.CPU)
	ret.Mem = newMemory()
	ret.Image = "boot"
	ret.State = CpuStopped
	ret.Breakpoints = make(map[uint16]int)

	return
}

func (ucpu *UserCpu) LoadHexFromRedis(key string) error {
	complete := make(chan error)

	ucpu.Comm <- func(c *UserCpu) { 
		c.redis.Comm <- func(r *RedisLand) {
			res, err := r.Conn.Do("GET", key)	
			if err == nil && res != nil {
				if err := msp43x.LoadHex(c.Mem, bufio.NewReader(bytes.NewBuffer(res.([]byte)))); err != nil {
					complete <- err
				} else {
					complete <- nil
				}
			} else {
				if err != nil {
					complete <- err
				} else {
					complete <- errors.New("key not found")
				}
			}

			return
		}
	
		return
	}

	return <- complete
}

func (ucpu *UserCpu) Loop() {
	var cpu *msp43x.CPU = ucpu.MCU

	for { 
		if ucpu.State == CpuRunning { 
			select {
			case req := <- ucpu.Comm:
				req(ucpu)

			default:
	 			ucpu.State = CpuRunning
	 		 	
	 			cur := cpu.Pc() // PC changes after Step, so remember it.

				nowStop := false		
		 	
				if kind, ok := ucpu.Breakpoints[cur]; ok { 
					switch kind {
					case BreakTrace:
					case BreakStep:
						nowStop = true
					case BreakStop:
						ucpu.State = CpuStopped
					}
				}
	
				if ucpu.State == CpuRunning {
		 			if err := cpu.Step(); err != nil {
		 				ucpu.State = CpuFault
		 			}

					if nowStop {
						ucpu.State = CpuStopped
					}
				}
			}
		} else {
			req := <- ucpu.Comm
			req(ucpu)
		}
	}
}
