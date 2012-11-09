package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"newmadrid/msp43x"
	cpuserver "newmadrid/cpuserver"
	"os"
	"strconv"
	"strings"
    "io/ioutil"
    "runtime"
//	"time"
)


type myHook struct { }
func (h myHook) ReadMemory(addr uint16, mem msp43x.Memory) (uint16,error) {
         fmt.Printf("Triggered addr %x\n", addr)
         return mem.LoadWordDirect(addr)
 }

func mustReader(path string) *bufio.Reader {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(fmt.Sprintf("open file \"%s\"", path))
		panic("eek")
	}

	r := bufio.NewReader(f)
	return r
}

// exists solely as an easy place to set a breakpoint
func ddd() int {
	i := 0
	return i
}

type devent map[uint16]bool

var (
	stabs *msp43x.Stabs
)

func setup(ca, cb, ma, mb, sa, db devent, istream *bool) {
	stabspath := flag.String("g", "", "path to stabs file")
	root := flag.String("r", "", "path to source root")

	cbfs := flag.String("C", "", "dump cpu before/after XXXXh")
	cafs := flag.String("c", "", "dump cpu after")
	mbfs := flag.String("M", "", "dump memory before/after")
	mafs := flag.String("m", "", "dump memory after")
	safs := flag.String("s", "", "sleep 2 seconds after")
	dbfs := flag.String("d", "", "debug break before")
	flag.BoolVar(istream, "i", false, "print instruction stream")

	flag.Parse()

	parse := func(m devent, str *string) {
		if str != nil && *str != "" {
			for _, saddr := range strings.Split(*str, ",") {
				if addr, err := strconv.ParseInt(saddr, 16, 32); err != nil {
					log.Printf("can't parse %s: %v", saddr, err)
				} else {
					m[uint16(addr)] = true
				}
			}
		}
	}

	parse(ca, cafs)
	parse(cb, cbfs)
	parse(ma, mafs)
	parse(mb, mbfs)
	parse(sa, safs)
	parse(db, dbfs)

	if *stabspath != "" {
		if *root == "" {
			*root = "/Users/thomas/codebase/msp430" // XXX
		}

		stabs = &msp43x.Stabs{
			Root: *root,
		}

		r := mustReader(fmt.Sprintf("%s/%s", *root, *stabspath))

		stabs.ReadStabs(r)
	}

}

func main() {

	// 	insns := []string{ 
	// 		"\x96\x45\x02\x00\x06\x00",
	// 		"\x86\x45\x00\x00",
	// 		"\xc8\x3c",
	// 		"\x00\x3e",
	// 		"\x09\x27",
	// 		"\x15\x10\x02\x00",
	// 		"\x36\x43",
	// 	}

	redis := flag.String("redis", "localhost:6379", "server:port redis connection")



	istream := false

	cpu_afters := make(devent)
	cpu_brackets := make(devent)
	memory_afters := make(devent)
	memory_brackets := make(devent)
	sleep_afters := make(devent)
	debug_befores := make(devent)

	setup(cpu_afters, cpu_brackets, memory_afters, memory_brackets, sleep_afters, debug_befores, &istream)

//	var cpu msp43x.CPU
//	var mem msp43x.SimpleMemory
	var err error


    // launch redis
	redisLand, err := cpuserver.NewRedisLand(*redis)
	if err != nil {
		log.Fatal("redis connect: ", err)
	}

	go redisLand.Loop()

    cpu := cpuserver.NewUserCpu("My CPU", redisLand)

    smem := new(msp43x.SimpleMemory)
//    cpu.MCU.SetMemory(smem)
    cpu.Mem = msp43x.NewHookableMemory(smem)
    cpu.MCU.SetMemory(cpu.Mem)

    go cpu.Loop()


	reader := mustReader("/tmp/boot.hex")
    data, err := ioutil.ReadAll(reader)
    if err != nil {
            log.Fatal("Error reading from file")
    }

    redisLand.Comm <- func(r *cpuserver.RedisLand) { 
        _,_ = r.Conn.Do("SET", "code", data)
    }

    err = cpu.LoadHexFromRedis("code")
    if err != nil {
		log.Fatal("parse ihex: ", err)
    }

//     cpu.Comm <- func(c *cpuserver.UserCpu) {
//         c.Breakpoints[0x4400] = 1
//     }
//
    
//     cpu.Comm <- func(c *cpuserver.UserCpu) {
//         c.Mem.ReadHook(0x437e, new(myHook))
//     }



    cpu.SetupDefaultHooks()
    cpu.Comm<-func(c *cpuserver.UserCpu) { 
            cpu.MCU.SetRegs([16]uint16{0x4400})
            cpu.State = cpuserver.CpuRunning
    }


    for { runtime.Gosched()}




//
//	maybe_cpu := func(cpu *msp43x.CPU, m devent, a uint16) {
//		if m[a] {
//			fmt.Printf("%v\n", cpu)
//		}
//	}
//
//	maybe_memory := func(mem *msp43x.SimpleMemory, m devent, a uint16) {
//		if m[a] {
//			log.Println(mem)
//		}
//	}
//
//	for {
//		var bytes []byte
//		var insn msp43x.Insn
//
//		var cur = cpu.Pc() // PC changes after Step, so remember it.
//
//		if debug_befores[cur] {
//			ddd()
//		}
//
//		if bytes, err = mem.Load6Bytes(cur); err != nil {
//			log.Fatal("decode insn: ", err)
//		}
//
//		if insn, err = msp43x.Disassemble(bytes); err != nil {
//			log.Fatal("decode insn: ", err)
//		}
//
//		maybe_cpu(&cpu, cpu_brackets, cur)
//		maybe_memory(&mem, memory_brackets, cur)
//
//		if stabs != nil && istream {
//			line := stabs.LineAt(cur)
//			if line != "" {
//				fmt.Println(line)
//			}
//		}
//
//		if istream {
//			fmt.Printf("%0.4x\t\t%v\n", cpu.Pc(), insn)
//		}
//
//		if err = cpu.Step(); err != nil {
//			log.Fatal("exec insn: ", err)
//		}
//
//		if stabs != nil {
//			fmt.Printf("%v\n", cpu)
//		}
//
//		if istream && cpu.Pc() != (cur+uint16(insn.Width)) {
//			fmt.Println("")
//		}
//
//		maybe_cpu(&cpu, cpu_brackets, cur)
//		maybe_memory(&mem, memory_brackets, cur)
//		maybe_cpu(&cpu, cpu_afters, cur)
//		maybe_memory(&mem, memory_afters, cur)
//
//		if sleep_afters[cur] {
//			time.Sleep(time.Duration(2) * time.Second)
//		}
//	}
}
