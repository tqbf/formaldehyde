package main

import (
	"canonical/goamz/ec2"
	"canonical/goamz/aws"
	"log"
	"flag"
	"strings"
	"fmt"
)

var MatasanoGroups = map[string]ec2.SecurityGroup{
	"Default": ec2.SecurityGroup{ "sg-96069eff", "default", },
	"Challenge": ec2.SecurityGroup{ "sg-c74148ae", "Wide open (DANGER)", },
}

var MatasanoAMIs = map[string]string { 
	"WebChallenge": "ami-ae3fc3c7",
	"ProtocolChallenge": "ami-69804f00",
}

var MatasanoAMIsReverse = map[string]string { 
	"ami-ae3fc3c7": "WebChallenge",
	"ami-69804f00": "ProtocolChallenge",

}

func nameTag(i *ec2.Instance) string {
	for _, t := range(i.Tags) { 
		if t.Key == "Name" {
			return t.Value
		}
	}
	return ""
}

func instancesMatching(filter string, instances []ec2.Instance) []ec2.Instance {
	ret := []ec2.Instance{}

	for _, i := range(instances) { 
		if strings.Contains(nameTag(&i), filter) {
			ret = append(ret, i)
		}
	}

	return ret
}

func instanceSummary(i *ec2.Instance) string {
	ami, ok := MatasanoAMIsReverse[i.ImageId]
	if !ok { 
		ami = i.ImageId
	}

	return fmt.Sprintf("%s: %s:%s (%s): %s", i.InstanceId, ami, nameTag(i), i.State.Name, i.DNSName)
}

type ActionFunc func(...string) (int, error)

func bulkAction(rs []ec2.Reservation, filter string, force bool, act ActionFunc) (int, error) {
	ids := []string{}

	for _, r := range(rs) { 
		insts := instancesMatching(filter, r.Instances)
		
		if len(insts) > 2 && !force  { 
			log.Fatalf("Can't stop more than 2 instances at a time without -force")
		}

		for _, i := range(insts) { 
			ids = append(ids, i.InstanceId)
		}
	}

	return act(ids...)
}

func main() { 
	filter := flag.String("filter", "", "only instances matching arg")
	list := flag.Bool("list", false, "list AMIs")
	stop := flag.Bool("stop", false, "stop AMIs matching filter")
	start := flag.Bool("start", false, "stop AMIs matching filter")
	reboot := flag.Bool("reboot", false, "stop AMIs matching filter")
	terminate := flag.String("terminate", "", "stop AMIs matching filter")
	force := flag.Bool("force", false, "force action")
	keypair := flag.String("keypair", "challenge ssh", "override key pair")
	aminame := flag.String("ami", "WebChallenge", "override ami")
	size := flag.String("size", "t1.micro", "override size")
	groupname := flag.String("group", "Challenge", "override security group")
	newname := flag.String("new", "", "new instance named ARG")

	flag.Parse()

	ami, ok := MatasanoAMIs[*aminame]
	if !ok {
		ami = *aminame
	}
	
	group, ok := MatasanoGroups[*groupname]
	if !ok { 
		log.Fatalf("Must provide a valid security group name")
	}	

	auth, err := aws.EnvAuth()
	if err != nil { 
		log.Fatalf("Can't read credentials from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY: %v", err)
	}
	
	me := ec2.New(auth, aws.USEast)

	instances, err := me.Instances(nil, nil)
	if err != nil { 
		log.Fatalf("Can't read instances from USEast: %v", err)
	}

	if *newname != "" { 
		resp, err := me.RunInstances(&ec2.RunInstances{
			ImageId: ami,
			MinCount: 1,
			MaxCount: 1,
			KeyName: *keypair,
			InstanceType: *size,
			SecurityGroups: []ec2.SecurityGroup{ group },
		})
		if err != nil {
			log.Fatalf("Can't start instance: %v", err)
		}

		iid := resp.Instances[0].InstanceId

		_, err = me.CreateTags([]string{ iid }, []ec2.Tag{
			ec2.Tag{
				Key: "Name",
				Value: *newname,
			},
		})
		if err != nil { 
			log.Fatalf("Can't tag instance: %v", err)
		}
	}

	if *stop && *filter != "" {
		r, err := bulkAction(instances.Reservations, *filter, *force, func(ids ...string) (int, error) { 
			r, e := me.StopInstances(ids...)
			if e != nil {
				return 0, e
			}

			return len(r.StateChanges), nil
		})
		if err != nil {
			log.Fatalf("Can't stop instance: %v", err)
		} else {
			log.Printf("Stopped %d instances", r)
		}
	}

	if *start && *filter != "" {
		r, err := bulkAction(instances.Reservations, *filter, *force, func(ids ...string) (int, error) { 
			r, e := me.StartInstances(ids...)
			if e != nil {
				return 0, e
			}

			return len(r.StateChanges), nil
		})
		if err != nil {
			log.Fatalf("Can't start instance: %v", err)
		} else {
			log.Printf("Started %d instances", r)
		}
	}

	if *reboot && *filter != "" {
		r, err := bulkAction(instances.Reservations, *filter, *force, func(ids ...string) (int, error) { 
			_, e := me.RebootInstances(ids...)
			if e != nil {
				return 0, e
			}

			return len(ids), nil
		})
		if err != nil {
			log.Fatalf("Can't reboot instance: %v", err)
		} else {
			log.Printf("Reboot %d instances", r)
		}
	}


	if *terminate != "" {
		res, err := me.TerminateInstances([]string{ *terminate })
		if err != nil { 
			log.Fatalf("Can't terminate instance: %v", err)
		}

		log.Printf("Terminated %d instances", len(res.StateChanges)) 
	}

	if *list { 
		for _, r := range(instances.Reservations) { 
			insts := r.Instances
			if *filter != "" { 
				insts = instancesMatching(*filter, insts)
			}
			for _, i := range(insts) { 
				fmt.Println(instanceSummary(&i))
			}
		}
	}
}