package main

import (
	"flag"
	"strings"
	"log"
	"fmt"	 
	"net/http"
	"crypto/tls"
	"io/ioutil"
	"os"
)

func main() { 
	concurrent := flag.Int("c", 10, "concurrent requests")
	pathfile := flag.String("paths", "urls.txt", "path file")
	hostfile := flag.String("hosts", "hosts.txt", "hosts file")

	flag.Parse()

	f, err := os.Open(*pathfile)
	if err != nil { 
		log.Fatalf("can't open paths: %v\n", err)
	}

	buf, _ := ioutil.ReadAll(f)
	paths := strings.Split(string(buf), "\n")

	f, err = os.Open(*hostfile)
	if err != nil { 
		log.Fatalf("can't open hosts: %v\n", err)
	}

	buf, _ = ioutil.ReadAll(f)
	hosts := strings.Split(string(buf), "\n")

	type hostpath struct {
		host string
		path string
	}

	incoming := make(chan hostpath)

	for i := 0; i < *concurrent; i++ {
		go func() { 
		        tr := &http.Transport{
		            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		        }
		        client := &http.Client{Transport: tr}

			for {
				hp, ok := <- incoming
				if !ok {
					break
				}

				res, _ := client.Get(fmt.Sprintf("%s%s", hp.host, hp.path))
				if res != nil { 
					if res.StatusCode >= 200 && res.StatusCode <= 299 {
						fmt.Printf("\n%s%s 200 %d bytes\n", hp.host, hp.path, res.ContentLength)
					}

					if res.StatusCode >= 500 && res.StatusCode <= 599 {
						fmt.Printf("\n%s%s 500 %d bytes\n", hp.host, hp.path, res.ContentLength)
					}

					if res.StatusCode >= 300 && res.StatusCode <= 399 {
						loc, ok := res.Header["Location"]
						if ok {
							fmt.Printf("\n%s%s redir %s\n", hp.host, hp.path, loc)
	
						}
					}

					res.Body.Close()
							
					fmt.Printf("%s%s %d                                                                  \r", hp.host, hp.path, res.StatusCode)
				}					
	
			}
		}()
	}

	for _, path := range(paths) { 
		for _, host := range(hosts) {
			incoming <- hostpath{
				host: host,
				path: path,
			}
		}
	}
	
	close(incoming)
}