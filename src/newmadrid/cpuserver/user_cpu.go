package main

import (
	"log"
	"newmadrid/msp43x"
	"bytes"
	"bufio"
	"errors"
	"fmt"
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
			cpu = NewUserCpu(req.Name,redis)
			cpus[req.Name] = cpu
			go cpu.Loop()
		}

		log.Printf("%p\n", cpu.MCU)

		req.Reply <- cpu
	}
}

func newMemory() *msp43x.HookableMemory {	
	return msp43x.NewHookableMemory(new(msp43x.SimpleMemory))
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

	Redis *RedisLand

	Breakpoints map[uint16]int	

	Comm chan CpuRequest

}

func (ucpu *UserCpu) SetupDefaultHooks() {
     // setup PrintChar
     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_USER_OUTPUT_BYTE, &WriteUserOutput{
                 cpu: ucpu,
                 })
     }

     // setup user input
     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_USER_INPUT_BUF_LENGTH, &ReadUserInputHook{
                 cpu: ucpu,
                 })
     }

     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_LOG_DEBUG_BUF_LENGTH, &DebugLogHook{
                 cpu: ucpu,
                 })
     }

     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_IO_ALARM, &AlarmHook{
                 cpu: ucpu,
                 })
     }

     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_IO_AIRFLOW, &AirflowHook{
                 cpu: ucpu,
                 })
     }

     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_IO_LOCK, &LockHook{
                 cpu: ucpu,
                 })
     }

     ucpu.Comm <- func(c *UserCpu) {
         c.Mem.WriteHook(M_IO_TEMPERATURE, &TemperatureHook{
                 cpu: ucpu,
                 })
     }
}

func NewUserCpu(cpuname string, redis *RedisLand) (ret *UserCpu) {
	ret = new(UserCpu)
	ret.MCU = new(msp43x.CPU)
	ret.Mem = newMemory()
	ret.Image = "boot"
	ret.State = CpuStopped
	ret.Comm = make(chan CpuRequest)
	ret.Breakpoints = make(map[uint16]int)
    ret.Name = cpuname
    ret.Redis = redis


	return
}


func (ucpu *UserCpu) LoadHexFromRedis(key string) error {
    fmt.Println("Loading memory from redis")
	complete := make(chan error)

	ucpu.Comm <- func(c *UserCpu) {
		c.Redis.Comm <- func(r *RedisLand) {
			res, err := r.Conn.Do("GET", key)
			if err == nil && res != nil {
				raw := res.([]byte)
				buf := bytes.NewBuffer(raw)

				if err := msp43x.LoadHex(c.Mem, bufio.NewReader(buf)); err != nil {
					complete <- err
				} else {
					complete <- nil
				}
//                fmt.Println(c.Mem)
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

	retval := <- complete
	fmt.Println("6")
	return retval
}

func (ucpu *UserCpu) Loop() {
	var cpu *msp43x.CPU = ucpu.MCU

	for { 
		if ucpu.State == CpuRunning { 
			select {
			case req := <-ucpu.Comm:
                fmt.Println(req)
				req(ucpu)

			default:
	 			ucpu.State = CpuRunning
	 		 	
	 			cur := cpu.Pc() // PC changes after Step, so remember it.
//                fmt.Printf("PC: %x\n",cur)

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
