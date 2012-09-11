package main

import (
	"log"
	"fmt"
	"newmadrid/msp43x"
)

func main() {
	insns := []string{ 
		"\x96\x45\x02\x00\x06\x00",
		"\x86\x45\x00\x00",
		"\xc8\x3c",
		"\x00\x3e",
		"\x09\x27",
		"\x15\x10\x02\x00",
		"\x36\x43",
	}

	for _, buf := range insns {
		insn, err := msp43x.Disassemble([]byte(buf))
		if err != nil {
			log.Fatal("disasm: ", err)
		}
		fmt.Printf("%v\n", insn)
	}
}