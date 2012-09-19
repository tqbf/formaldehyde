package msp43x

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"os"
	"strings"
)

type stab struct {
	addr uint16
	file uint
	line int
}

type Stabs struct {
	maxfile uint
	files map[uint]*string	
	files_rev map[string]uint
	stabs map[uint16]stab

	Root string
}

var (
	line_rx = regexp.MustCompile("/\\*\\s+file\\s+(.*?)\\s+line\\s+(\\d+)\\s+addr\\s+(.*?)\\s+\\*/")
)

func lines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}	

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil
	}

	return strings.Split(string(buf), "\n")
}

func (stabs *Stabs) LineAt(addr uint16) string {
	if stab, ok := stabs.stabs[addr]; ok {
		if path, ok := stabs.files[stab.file]; ok {
			slines := lines(fmt.Sprintf("%s/%s", stabs.Root, *path))
			
			if slines != nil && (stab.line-1) < len(slines) {
				return slines[stab.line-1]
			}
		}
	} 
	
	return ""
}

func (stabs *Stabs) ReadStabs(in *bufio.Reader) int {
	stabs.files = make(map[uint]*string)
	stabs.files_rev = make(map[string]uint)
	stabs.stabs = make(map[uint16]stab)

	for { 
		line, _, xerr := in.ReadLine()
		if xerr != nil { 
			return len(stabs.stabs)
		}
	
		m := line_rx.FindSubmatch(line)

		if m != nil {
			filename := string(m[1])
			line, e1 := strconv.ParseInt(string(m[2]), 0, 32)
			addr, e2 := strconv.ParseInt(string(m[3]), 0, 32)

			if e1 == nil && e2 == nil {
				var (
					fileid	uint
					ok bool
				) 
				if fileid, ok = stabs.files_rev[filename]; !ok {
					fileid = stabs.maxfile
					stabs.maxfile += 1	

					stabs.files[fileid] = &filename
					stabs.files_rev[filename] = fileid	
				}

				stabs.stabs[uint16(addr)] = stab{
					addr: uint16(addr),
					file: fileid,
					line: int(line),
				}
			}
		}
	}

	return len(stabs.stabs)
}