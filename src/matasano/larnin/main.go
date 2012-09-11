package main

import (
  "fmt"
)

func main() { 
	fmt.Printf("Helu\n")

	s0 := uint16(0xefbf)
	fmt.Printf("%d %d\n", s0, ^s0)

	s1 := uint32(s0)
	fmt.Printf("%d %d\n", s1, ^s1)

	s2 := int16(s0)
	fmt.Printf("%d %d\n", s2, ^s2)

	var src, tmp uint16
	fmt.Println("RRC")
	src = 43690
	sbit := 1
	x := src & 1
	tmp = uint16(sbit << 15)
	tmp |= src >> 1
	fmt.Printf("%b\n", src)
	fmt.Printf("%b\n", tmp)
	fmt.Printf("%d\n", x)
	src = tmp
	sbit = 0 
	x = src & 1
	tmp = uint16(sbit << 15)
	tmp |= src >> 1
	fmt.Printf("%b\n", src)
	fmt.Printf("%b\n", tmp)
	fmt.Printf("%d\n", x)

	fmt.Println("SWPB")
	tmp = (src >> 8) | ((src & 0xff) << 8)
	fmt.Printf("%x\n", src)	
	fmt.Printf("%x\n", tmp)	

	fmt.Println("RRA")
	
	src = 0x8000
	x = src & 1
	y := (src >> 15) & 1
	tmp = src >> 1
	tmp = uint16((tmp & 0x4000) | (y << 15))

	fmt.Printf("%x\n", src)
	fmt.Printf("%x\n", tmp)

	fmt.Println("SXT")
	
	src = 0x80
	if src & 0x80 != 0 {
		tmp = (0xff<<8) | src
	} else {
		tmp = src & 0xff
	}

	fmt.Printf("%x\n", src)
	fmt.Printf("%x\n", tmp)

	src = 0x7f
	if src & 0x80 != 0 {
		tmp = (0xff<<8) | src
	} else {
		tmp = src & 0xff
	}

	fmt.Printf("%x\n", src)
	fmt.Printf("%x\n", tmp)

	src = 0x8000
	fmt.Printf("%d %d\n", uint16(src), int16(src))
	src = 0xc000
	fmt.Printf("%d %d\n", uint16(src), int16(src))
	src = 0xc00
	fmt.Printf("%d %d\n", uint16(src), int16(src))

	src = 0x1f0
	dst := uint16(0x2f0)
	fmt.Printf("%d %d\n", (src + dst), uint8(src) + uint8(dst))

	src = 5
	dst = 10
	tmp = dst + ((^src) + 1)

	fmt.Printf("%x\n", src)
	fmt.Printf("%x\n", dst)
	fmt.Printf("%x\n", tmp)

	src = 0x0040
	dst = 0xffff
	tmp = (^src) & dst
	fmt.Printf("%b\n", tmp)

	tmp = uint16(uint8(^src) & uint8(dst))
	fmt.Printf("%b\n", tmp)
		
	src = 0xaaaa
	dst = 0xffff
	tmp = src ^ dst
	fmt.Printf("%b\n", tmp)		


}
