package matasano

import (
	"os"
	"bytes"
	"io"
)

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