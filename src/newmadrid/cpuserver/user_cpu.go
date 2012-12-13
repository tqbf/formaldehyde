package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"newmadrid/msp43x"
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
			cpu = NewUserCpu(req.Name, redis)
			cpus[req.Name] = cpu
			go cpu.Loop()
		}

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

func NewUserCpu(cpuname string, redis *RedisLand) (ret *UserCpu) {
	ret = new(UserCpu)
	ret.MCU = new(msp43x.CPU)
	ret.Mem = newMemory()
	ret.MCU.SetMemory(ret.Mem)
	ret.MCU.SetRegs([16]uint16{0x4400})
	ret.Image = "boot"
	ret.State = CpuStopped
	ret.Comm = make(chan CpuRequest)
	ret.Breakpoints = make(map[uint16]int)
	ret.Name = cpuname
	ret.Redis = redis

	// setup PrintChar
	ret.Mem.WriteHook(M_USER_OUTPUT_BYTE, WriteUserOutputHook(ret))
	// setup user input
	ret.Mem.WriteHook(M_USER_INPUT_BUF_LENGTH, ReadUserInputHook(ret))
	ret.Mem.WriteHook(M_LOG_DEBUG_BUF_LENGTH, DebugLogHook(ret))
	ret.Mem.WriteHook(M_IO_ALARM, AlarmHook(ret))
	ret.Mem.WriteHook(M_IO_AIRFLOW, AirflowHook(ret))
	ret.Mem.WriteHook(M_IO_LOCK, LockHook(ret))
	ret.Mem.WriteHook(M_IO_TEMPERATURE, TemperatureHook(ret))

	return
}

func (ucpu *UserCpu) LoadHexFromRedis(key string) error {
	fmt.Printf("Loading hex image from redis:%s\n", key)

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

	retval := <-complete
	fmt.Println("complete")
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
			req := <-ucpu.Comm
			req(ucpu)
		}
	}
}
