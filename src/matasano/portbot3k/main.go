package main

import (
	"flag"
	"io/ioutil"
	"log"
	"matasano/util"
	"net"
	"os"
	"strings"
	"runtime"
	"syscall"
	"fmt"	 
	"time"   

	_ "github.com/bmizerany/pq"
	"database/sql"
)

func parseDatabase(cs string) []string { 
	var dbconfig []string

	tups := strings.SplitN(cs, "@", 2)
	if len(tups) > 1 {
		user := tups[0]
		u_p := strings.SplitN(user, ":", 2)
		if len(u_p) > 1 {
			dbconfig = append(dbconfig, fmt.Sprintf("user=%s", u_p[0]))
			dbconfig = append(dbconfig, fmt.Sprintf("password=%s", u_p[1]))
		} else {
			dbconfig = append(dbconfig, fmt.Sprintf("user=%s", u_p[0]))
		}
		
		tups = strings.SplitN(tups[1], "/", 2)
		if len(tups) > 1 { 
			dbconfig = append(dbconfig, fmt.Sprintf("dbname=%s", tups[1]))

			tups = strings.SplitN(tups[0], ":", 2)
			if len(tups) > 1 {
				dbconfig = append(dbconfig, fmt.Sprintf("host=%s", tups[0]))
				dbconfig = append(dbconfig, fmt.Sprintf("port=%s", tups[1]))
			} else {
				dbconfig = append(dbconfig, fmt.Sprintf("host=%s", tups[0]))
				dbconfig = append(dbconfig, fmt.Sprintf("port=5432"))
			}		
		} else {
			log.Fatalf("Bad db string \"%s\": expected user@host/db, but no db", cs)
		}
	} else {
		log.Fatalf("Bad db string \"%s\": expected user@host/db, but no user", cs)
	}

	return dbconfig
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Cur: 8192,
		Max: 8192,
	})

	ip_in_f := os.Stdin
	infile := flag.String("hosts", "", "file containing IP addresses (default: stdin)")
	max_inflight := flag.Int("max", 10, "maximum in flight requests")
	dbconfstr := flag.String("postgres", "", "user[:pw]@host[:port]/database")
	provision := flag.Bool("provision", false, "set up tables")

	flag.Parse()

	var (
		db *sql.DB
	)

	policy := util.DefaultPolicy{
		MaxInFlight: *max_inflight,
	}

	if *dbconfstr != "" {
		var err error

		db, err = sql.Open("postgres", strings.Join(parseDatabase(*dbconfstr), " "))
		if err != nil { 
			log.Fatalf("Can't open postgres: %v", err)
		} 

		policy.LiveResults = make(chan util.ProbePositive, 20)
	}

	if *infile != "" {
		var err error
		ip_in_f, err = os.Open(*infile)
		if err != nil { 
			log.Fatalf("Can't open \"%s\": %v", infile, err)
		}
	}

	if *provision { 
		r, e := db.Exec(`
			CREATE TABLE portbot_positive_ports (
				id SERIAL PRIMARY KEY,
				time_added INT NOT NULL,
				time_elapsed INT,
				host VARCHAR(100),
				port INT
			);`)
		if e != nil { 
			log.Fatalf("Can't create database: %v", e)
		} else {
			log.Printf("Database created, result: %v", r)
			os.Exit(0)	
		}
	}

	rset := util.ParsePortRanges(flag.Arg(0))

	buf, err := ioutil.ReadAll(ip_in_f)
	if err != nil {
		log.Fatal("can't read IP input file")
	}

	var addrs []*net.IPAddr

	for _, line := range strings.Split(string(buf), "\n") {
		line = strings.Trim(line, " \t")

		_, cidr, err := net.ParseCIDR(line)
		if err == nil { 
			top, bottom := util.IpRange(*cidr)
			for top = top + 1; top < bottom; top++ {
				addrs = append(addrs, &net.IPAddr{
					IP: util.ScalarToIp(top),
				})
			}

			continue
		}

		if addr, err := net.ResolveIPAddr("ip4", line); err != nil {
			log.Printf("invalid IP address \"%s\" (continuing)", line)
			continue
		} else {
			addrs = append(addrs, addr)
		}
	}

	if db == nil { 
		result := util.PortScan(addrs, rset, &policy)
		for sockAddr, _ := range(result) {
			log.Printf("%s\n", sockAddr)
		}
	} else {
		finished := make(chan bool)

		go util.PortScan(addrs, rset, &policy)		
		go func() {
			var batch []util.ProbePositive

			pushQuery := func() { 
				var rows []string
				t := time.Now()					

				base := "INSERT INTO portbot_positive_ports ( time_added, time_elapsed, host, port ) VALUES"
				
				for _, positive := range(batch) { 
					rows = append(rows, fmt.Sprintf("( %d, %d, '%s', %d )", 
						uint64(t.Unix()),
						positive.Elapsed,
						positive.SockAddr.Addr,
						positive.SockAddr.Port))
				}
	 
				if len(batch) > 0 { 
					_, err := db.Exec(fmt.Sprintf("%s %s;", base, strings.Join(rows, ", ")))
					if err != nil { 
						log.Printf("insert failed: %v (%s %s;)\n", err, base, strings.Join(rows, ", "))
					}

					batch = batch[:0]			
				}
			}

			for { 
				select { 
		 		case pp, ok := <- policy.LiveResults:
					if !ok {
						pushQuery()
						os.Exit(0)
					}
 
		 			log.Printf("%s\n", pp.SockAddr)
		 			batch = append(batch, pp)
		 		case <- time.After(5 * time.Second): 
					pushQuery()
		 		}
			}
		}()

		_ = <- finished
	} 
}
