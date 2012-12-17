package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"newmadrid/msp43x"
	"time"
)

// memory hooks, I/O addresses
const (
	// REDIS-BASED USER I/O
	M_USER_INPUT_BUF_ADDRESS = 0x280
	M_USER_INPUT_BUF_LENGTH  = 0x282
	M_USER_OUTPUT_BYTE       = 0x5ce

	// "REAL" I/O (should this also be logged to redis?)
	M_IO_LOCK        = 0x220 // done
	M_IO_ALARM       = 0x222 // done
	M_IO_TEMPERATURE = 0x224 // done
	M_IO_AIRFLOW     = 0x226 // done

	// REDIS-BASED LOGGING
	M_LOG_DEBUG_BUF_LOCATION   = 0x260 // done
	M_LOG_DEBUG_BUF_LENGTH     = 0x262 // done
	M_LOG_VISITOR_BUF_LOCATION = 0x264
	M_LOG_VISITOR_BUF_LENGTH   = 0x266
)

func (c UserCpu) DebugLog(entry *bytes.Buffer) {
	max_length_of_entry := 100

	rediskey := fmt.Sprintf("%s:debuglog", c.Name)

	if entry.Len() > max_length_of_entry {
		entry.Truncate(max_length_of_entry)
	}

	c.Redis.Comm <- func(r *RedisLand) {
		r.Conn.Do("LPUSH", rediskey, fmt.Sprintf("%d:%s", time.Now().Unix(), entry.String()))
		r.Conn.Do("LTRIM", rediskey, 0, 100)
	}
}

func ReadUserInputHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, length uint16, mem msp43x.Memory) (err error) {
		// load destination address
		dst, err := mem.LoadWordDirect(M_USER_INPUT_BUF_ADDRESS)
		if err != nil {
			return
		}

		var raw []byte
		userinput := make(chan []byte)

		state := cpu.State
		cpu.State = CpuIoSleep
		for {
			// load from redis
			cpu.Redis.Comm <- func(r *RedisLand) {
				rediskey := fmt.Sprintf("%s:input", cpu.Name)

				res, err := r.Conn.Do("GET", rediskey)
				if err != nil {
					return
				}

				if res != nil {
					r.Conn.Do("DEL", rediskey)
					raw := res.([]byte)
					log.Printf("Importing %v bytes (total was %v)", length, len(raw))
					userinput <- raw
				} else {
					userinput <- nil
				}
			}
			raw = <-userinput
			if raw != nil {
				break
			}

			time.Sleep(500 * time.Millisecond)
		}
		cpu.State = state

		// store a max of  value bytes at address in 0x5d0
		if uint16(len(raw)) < length {
			length = uint16(len(raw))
		}

		// store a max of  value bytes at address in 0x5d0
		if uint16(len(raw)) < length {
			length = uint16(len(raw))
		}

		for i := uint16(0); i < length; i++ {
			mem.StoreByte(dst+i, raw[i])
		}

		return mem.StoreWordDirect(addr, length)
	}
}

func DebugLogHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, value uint16, mem msp43x.Memory) (err error) {
		loc, err := mem.LoadWordDirect(M_LOG_DEBUG_BUF_LOCATION)
		if err != nil {
			return
		}

		data, err := mem.Read(loc, value)
		if err != nil {
			return
		}

		cpu.DebugLog(bytes.NewBuffer(data))

		return
	}
}

func AlarmHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, value uint16, mem msp43x.Memory) (err error) {
		if err = mem.StoreWordDirect(addr, value); err != nil {
			return
		}

		cpu.Redis.Comm <- func(r *RedisLand) {
			r.Conn.Do("SET", fmt.Sprintf("%s:alarm", cpu.Name), value)
		}

		if value == 0 {
			cpu.DebugLog(bytes.NewBufferString("ALARM DISARMED"))
		} else {
			cpu.DebugLog(bytes.NewBufferString("ALARM ARMED"))
		}

		return
	}
}

func LockHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, value uint16, mem msp43x.Memory) (err error) {
		if err = mem.StoreWordDirect(addr, value); err != nil {
			return
		}

		cpu.Redis.Comm <- func(r *RedisLand) {
			r.Conn.Do("SET", fmt.Sprintf("%s:lock", cpu.Name), value)
		}

		if value == 0 {
			cpu.DebugLog(bytes.NewBufferString("LOCK ENGAGED"))
		} else {
			cpu.DebugLog(bytes.NewBufferString("LOCK DISENGAGED"))
		}

		return
	}
}

func TemperatureHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, value uint16, mem msp43x.Memory) (err error) {
		if err = mem.StoreWordDirect(addr, value); err != nil {
			return
		}

		cpu.Redis.Comm <- func(r *RedisLand) {
			r.Conn.Do("SET", fmt.Sprintf("%s:temperature", cpu.Name), value)
		}

		cpu.DebugLog(bytes.NewBufferString(fmt.Sprintf("TEMPERATURE TARGET: %d", uint16(value))))

		return
	}
}

func AirflowHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, value uint16, mem msp43x.Memory) (err error) {
		if err = mem.StoreWordDirect(addr, value); err != nil {
			return
		}

		cpu.Redis.Comm <- func(r *RedisLand) {
			r.Conn.Do("SET", fmt.Sprintf("%s:airflow", cpu.Name), value)
		}

		cpu.DebugLog(bytes.NewBufferString(fmt.Sprintf("AIRFLOW TARGET: %d", byte(value))))

		return
	}
}

func WriteUserOutputHook(cpu *UserCpu) msp43x.WriteHookFunc {
	return func(addr, value uint16, mem msp43x.Memory) (err error) {
		curOut := make(chan []byte)

		// load from redis
		cpu.Redis.Comm <- func(r *RedisLand) {
			res, err := r.Conn.Do("GETRANGE", fmt.Sprintf("%s:output", cpu.Name), -399, 400)
			if err != nil {
				return
			}

			if res == nil {
				//return mem.StoreWordDirect(addr, 0)
				err = errors.New("key not found")
				return
			}

			raw := res.([]byte)
			curOut <- raw
		}

		raw := <-curOut

		var output bytes.Buffer
		output.Write(raw)
		output.WriteByte(byte(value))

		cpu.Redis.Comm <- func(r *RedisLand) {
			r.Conn.Do("SET", fmt.Sprintf("%s:output", cpu.Name), output.String())
		}

		return
	}
}
