package main

import (
        "newmadrid/msp43x"
//        "fmt"
        "log"
        "bytes"
)

//    0x5d0: // buffer address
//         0x5d2 // length triggers. update after write

type ReadUserInputHook struct { 
        cpu *UserCpu
        }

        func (h ReadUserInputHook) WriteMemory(addr, length uint16, mem msp43x.Memory) error {
        // load destination address
        dst, err := mem.LoadWordDirect(0x5d0)
        if err != nil {
                log.Fatal("Error")

        }

        userinput := make(chan []byte)

        // load from redis
        h.cpu.Redis.Comm <- func(r *RedisLand) { 
            var rediskey bytes.Buffer
            rediskey.WriteString(h.cpu.Name)
            rediskey.WriteString("-input")


            res, err := r.Conn.Do("GET", rediskey.String())
            if err != nil {
                    log.Fatal("Could not get user input from redis.")
            }

            if res == nil {
                    log.Println("Redis key not found.")
//                    return mem.StoreWordDirect(addr,uint16(0))
                     return
            }

            raw := res.([]byte)
            log.Printf("Importing %v bytes (total was %v)", length, len(raw))
            userinput<-raw
        }
        raw := <-userinput
        // store a max of  value bytes at address in 0x5d0
        if uint16(len(raw)) < length {
                length = uint16(len(raw))
        }

        var i uint16;
        for i = 0; i < length; i++ {
                mem.StoreByte(dst+i, raw[i])
        }

        return mem.StoreWordDirect(addr, length)
 }


// address 0x5ce
type WriteUserOutput struct {
        cpu *UserCpu
        }

func (h WriteUserOutput) WriteMemory(addr, value uint16, mem msp43x.Memory) error {
        curOut := make(chan []byte)

        var rediskey bytes.Buffer
        rediskey.WriteString(h.cpu.Name)
        rediskey.WriteString("-output")
        // load from redis
        h.cpu.Redis.Comm <- func(r *RedisLand) { 

            res, err := r.Conn.Do("GETRANGE", rediskey.String(), -399, 400)
            if err != nil {
                    log.Fatal("Could not get current user output from redis.")
            }

            if res == nil {
                    log.Println("Redis key not found.")
                    //return mem.StoreWordDirect(addr, 0)
                    return
            }
            raw := res.([]byte)

            curOut<-raw

        }
        raw := <-curOut

        var output bytes.Buffer
        output.Write(raw)
        output.WriteByte(byte(value))

        h.cpu.Redis.Comm <- func(r *RedisLand) { 
        _,_ = r.Conn.Do("SET", rediskey.String(), output.String())
        }

        return nil
 }
