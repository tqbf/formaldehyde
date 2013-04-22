package util

import (
	"os"
	"bytes"
	"io"
	"net"
	"fmt"
)

func IpToScalar(ip net.IP) uint32 {
  raw := ip.To4()
  return uint32(raw[0]) << 24 | uint32(raw[1]) << 16 | uint32(raw[2]) << 8 | uint32(raw[3])
}

func ScalarToIp(ip uint32) net.IP {
  // can't find a better way to do this
  return net.ParseIP(fmt.Sprintf("%d.%d.%d.%d",
                ((ip>>24)&0xff),
                ((ip>>16)&0xff),
                ((ip>>8)&0xff),
                ((ip>>0)&0xff)))
}

func IpRange(cidr net.IPNet) (uint32, uint32) {
  b, _ := cidr.Mask.Size()
  
  mask := (uint32(1) << uint32(b)) - uint32(1)
  mask <<= uint32(32) - uint32(b)

  base := IpToScalar(cidr.IP) & mask
  top := IpToScalar(cidr.IP) | ^mask
  return base, top
}

func Slurp(filename string) (*bytes.Buffer, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	var (
		tmpbuf [1024]byte
		l int
	)

	for err = nil; err == nil;  {
		l, err = f.Read(tmpbuf[0:])
		if(err != nil && err != io.EOF) {
			return nil, err
		} 

		buf.Write(tmpbuf[0:l])
	}

	return buf, nil
}

func Barf(filename string, data []byte) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	defer f.Close()
	
	off := 0
	l := 0
	for err = nil; err == nil && off < len(data);  {
		l, err = f.Write(data[off:])
		if(err != nil && err != io.EOF) {
			return err
		} 

		off += l
	}

	return nil
}

