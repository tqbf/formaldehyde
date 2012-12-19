package msp43x

import (
	"bufio"
	"strconv"
	"errors"
//    "fmt"
)

func LoadHex(memory Memory, in *bufio.Reader) (err error) {
	err = nil

	memory.Clear()

	for { 
		line, _, xerr := in.ReadLine()
		if xerr != nil { 
			return
		}

		if len(line) < 3 {
			return errors.New("line too short")
		}		

		var (
			bytelen int64
			baseaddr int64
			addr uint16
		)

		if bytelen, err = strconv.ParseInt(string(line[1:3]), 16, 8); err != nil {
			return
		}
		
		if len(line) < 1 + 2 + 2 + 4 + (int(bytelen) * 2) + 2 {
			return errors.New("line too short for len")
		}

		if baseaddr, err = strconv.ParseInt(string(line[3:7]), 16, 32); err != nil {
			return
		}

		addr = uint16(baseaddr)

		for i := 0; i < int(bytelen); i++ {
			var byte int64
			
			if byte, err = strconv.ParseInt(string(line[9 + (i * 2):9 + (i * 2) + 2]), 16, 16); err != nil {
				return
			}	

//            fmt.Printf("Storing %v at addr %x\n", uint8(byte), addr)
			if err = memory.StoreByte(addr, uint8(byte)); err != nil {
				return
			}
			addr += 1
		}
	}

	return
}
