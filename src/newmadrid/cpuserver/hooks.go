package main

import (
	"newmadrid/msp43x"
	"fmt"
	"bytes"
	"log"
	"errors"
)

const (
	InputAddr = 0x5d0
	InputLength = 0x5d2	// MAGIC: triggers update after write
	Output = 0x5ce
)

type ReadUserInputHook struct {
	cpu *UserCpu
}

func (h ReadUserInputHook) WriteMemory(addr, length uint16, mem msp43x.Memory) (err error) {
	// load destination address
	dst, err := mem.LoadWordDirect(InputAddr)
	if err != nil {
		return
	}

	userinput := make(chan []byte)

	// load from redis
	h.cpu.Redis.Comm <- func(r *RedisLand) {
		rediskey := fmt.Sprintf("%s:input", h.cpu.Name)

		res, err := r.Conn.Do("GET", rediskey)
		if err != nil {
			return
		}

		if res == nil {
			err = errors.New("key not found")
			return
		}

		raw := res.([]byte)
		log.Printf("Importing %v bytes (total was %v)", length, len(raw))
		userinput <- raw
	}
	raw := <-userinput

	// store a max of  value bytes at address in 0x5d0
	if uint16(len(raw)) < length {
		length = uint16(len(raw))
	}

	for i := uint16(0); i < length; i++ {
		mem.StoreByte(dst+i, raw[i])
	}

	return mem.StoreWordDirect(addr, length)
}

// address 0x5ce
type WriteUserOutput struct {
	cpu *UserCpu
}

func (h WriteUserOutput) WriteMemory(addr, value uint16, mem msp43x.Memory) (err error) {
	curOut := make(chan []byte)

	rediskey := fmt.Sprintf("%s:output", h.cpu.Name)

	// load from redis
	h.cpu.Redis.Comm <- func(r *RedisLand) {
		res, err := r.Conn.Do("GETRANGE", rediskey, -399, 400)
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

	h.cpu.Redis.Comm <- func(r *RedisLand) {
		_, _ = r.Conn.Do("SET", rediskey, output.String())
	}

	return nil
}
