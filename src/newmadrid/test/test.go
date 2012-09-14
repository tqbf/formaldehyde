package main

import (
	"os"
	"bufio"
	"log"
	"fmt"
	"newmadrid/msp43x"
)

func kweep() int { 
	i := 0
	return i
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

	var cpu msp43x.CPU
	var mem msp43x.SimpleMemory

	f, err := os.Open("/tmp/boot.hex")
	if err != nil { 
		log.Fatal("open ihex file")
	}	

	r := bufio.NewReader(f)

	if err = msp43x.LoadHex(&mem, r); err != nil {
		log.Fatal("parse ihex: ", err)
	}

	log.Println(&mem)

	cpu.SetMemory(&mem)
	cpu.SetRegs([16]uint16{ 0x4400 })

	for {
		var bytes []byte
		var insn msp43x.Insn

		if cpu.Pc() == 0xf480 {
			kweep()
		}

		if bytes, err = mem.Load6Bytes(cpu.Pc()); err != nil {
			log.Fatal("decode insn: ", err)
		}
		
		if insn, err = msp43x.Disassemble(bytes); err != nil {
			log.Fatal("decode insn: ", err)
		}

		fmt.Println(insn)

		if err = cpu.Step(); err != nil {
			log.Fatal("exec insn: ", err)
		}

		fmt.Printf("%v\n", cpu)		
	}
}