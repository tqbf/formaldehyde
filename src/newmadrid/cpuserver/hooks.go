package main

import (
        "newmadrid/msp43x"
//        "fmt"
        "log"
        "bytes"
        "time"
//        "encoding/binary"
)

// memory hooks, I/O addresses
const ( 
    // REDIS-BASED USER I/O
    M_USER_INPUT_BUF_ADDRESS = 0x280
    M_USER_INPUT_BUF_LENGTH = 0x282
    M_USER_OUTPUT_BYTE = 0x5ce

    // "REAL" I/O (should this also be logged to redis?)
    M_IO_LOCK = 0x220                    // done
    M_IO_ALARM = 0x222                   // done
    M_IO_TEMPERATURE = 0x224             // done
    M_IO_AIRFLOW = 0x226                 // done

    // REDIS-BASED LOGGING
    M_LOG_DEBUG_BUF_LOCATION = 0x260     // done
    M_LOG_DEBUG_BUF_LENGTH = 0x262       // done
    M_LOG_VISITOR_BUF_LOCATION = 0x264
    M_LOG_VISITOR_BUF_LENGTH = 0x266
    )

func (c UserCpu) DebugLog(entry *bytes.Buffer) {
        max_length_of_entry := 100

        var rediskey bytes.Buffer
        rediskey.WriteString(c.Name)
        rediskey.WriteString("-debuglog")

        if entry.Len() > max_length_of_entry {
            entry.Truncate(max_length_of_entry)
        } 

//        fmt.Println(entry.String())
        c.Redis.Comm <- func(r *RedisLand) { 
            r.Conn.Do("LPUSH", rediskey.String(), entry.String())
            r.Conn.Do("LTRIM", rediskey.String(), 0, 100 )
        }

}

type ReadUserInputHook struct { 
        cpu *UserCpu
        }

func (h ReadUserInputHook) WriteMemory(addr, length uint16, mem msp43x.Memory) error {
        // load destination address
        dst, err := mem.LoadWordDirect(M_USER_INPUT_BUF_ADDRESS)
        if err != nil {
                log.Fatal("Error")

        }
        
        var raw []byte
        userinput := make(chan []byte)
        for {
            // load from redis
            h.cpu.Redis.Comm <- func(r *RedisLand) { 
                var rediskey bytes.Buffer
                rediskey.WriteString(h.cpu.Name)
                rediskey.WriteString("-input")
    
                res, err := r.Conn.Do("GET", rediskey.String())
                if err != nil {
                    log.Fatal("Could not get user input from redis.")
                }
    
                if res != nil {
                        r.Conn.Do("DEL", rediskey.String())
                        raw := res.([]byte)
                        log.Printf("Importing %v bytes (total was %v)", length, len(raw))
                        userinput<-raw
                } else {
                        userinput<-nil
                }
            }
            raw = <-userinput
            if raw != nil {
                    break
            }
    
            time.Sleep(500 * time.Millisecond)
        }

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


type DebugLogHook struct {
        cpu *UserCpu
}

func (h DebugLogHook) WriteMemory(addr, value uint16, mem msp43x.Memory) error {

        loc, err := mem.LoadWordDirect(M_LOG_DEBUG_BUF_LOCATION)
        if err != nil {
                log.Fatal("Error")
        }

        data, err := mem.Read(loc, value)
        if err != nil {
                log.Fatal("Error")
        }

        var output bytes.Buffer
        output.Write(data)
        h.cpu.DebugLog(&output)


        return nil
}

type AlarmHook struct {
        cpu *UserCpu
}

func (h AlarmHook) WriteMemory(addr, value uint16, mem msp43x.Memory) error {

    err := mem.StoreWordDirect(addr, value)

    var rediskey bytes.Buffer
    rediskey.WriteString(h.cpu.Name)
    rediskey.WriteString("-alarm")

    h.cpu.Redis.Comm <- func(r *RedisLand) { 
        _,_ = r.Conn.Do("SET", rediskey.String(), value)
    }

    var entry bytes.Buffer
    if(value == 0) {
            entry.WriteString("Alarm has been disabled")
    } else {
            entry.WriteString("Alarm has been activated")
    }
    h.cpu.DebugLog(&entry)

    return err
}


type LockHook struct {
        cpu *UserCpu
}

func (h LockHook) WriteMemory(addr, value uint16, mem msp43x.Memory) error {

    err := mem.StoreWordDirect(addr, value)

    var rediskey bytes.Buffer
    rediskey.WriteString(h.cpu.Name)
    rediskey.WriteString("-lock")

    h.cpu.Redis.Comm <- func(r *RedisLand) { 
        _,_ = r.Conn.Do("SET", rediskey.String(), value)
    }

    var entry bytes.Buffer
    if(value == 0) {
            entry.WriteString("The lock has been locked ")
    } else {
            entry.WriteString("The Lock has been unlocked")
    }
    h.cpu.DebugLog(&entry)

    return err
}


type TemperatureHook struct {
        cpu *UserCpu
}

func (h TemperatureHook) WriteMemory(addr, value uint16, mem msp43x.Memory) error {

    err := mem.StoreWordDirect(addr, value)

    var rediskey bytes.Buffer
    rediskey.WriteString(h.cpu.Name)
    rediskey.WriteString("-temperature")

    h.cpu.Redis.Comm <- func(r *RedisLand) { 
        _,_ = r.Conn.Do("SET", rediskey.String(), value)
    }

    var entry bytes.Buffer
    entry.WriteString("The temperature has been set to ")
    entry.WriteString(string(uint16(value)))
    h.cpu.DebugLog(&entry)

    return err
}


type AirflowHook struct {
        cpu *UserCpu
}

func (h AirflowHook) WriteMemory(addr, value uint16, mem msp43x.Memory) error {

    err := mem.StoreWordDirect(addr, value)

    var rediskey bytes.Buffer
    rediskey.WriteString(h.cpu.Name)
    rediskey.WriteString("-airflow")

    h.cpu.Redis.Comm <- func(r *RedisLand) { 
        _,_ = r.Conn.Do("SET", rediskey.String(), value)
    }


    var entry bytes.Buffer
    entry.WriteString("The airflow has been set to ")
    entry.WriteByte(byte(value))
    h.cpu.DebugLog(&entry)

    return err
}


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
