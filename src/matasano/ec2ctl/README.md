# ec2ctl - manage challenge instances

## Installing

You can ask me for an OS X binary, or:

1. Install go and set your GOROOT variable so that "go help" works.

2. Clone the whole tree this project is in (from the root) and set GOPATH to point to it.

3. "go install matasano/ec2ctl"

4. Run your binary out of bin/ec2ctl

This also works on 64 bit Linux.

Populate env AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY with your key ID and key value 
(or run the command with "env").

## Usage

*ec2ctl -list*

List all our USEast instances.

*ec2ctl -list BOB*

List all instances tagged with the name "*BOB*".

*ec2ctl -new BILL_Q_WEB*

Create a new web challenge for BILL_Q.

*ec2ctl -new BILL_Q_PROTO -ami ProtocolChallenge*

Create a new protocol challenge. Type *ec2ctl -help* for other options for creating
intances; you can use any valid AMI and Security Group.

*ec2ctl -stop -filter BOB*

Stop all Bob's instances. If there are more than 2, add *-force*.

*ec2ctl -start -filter BOB*

Start all Bob's instances. If there are more than 2, add *-force*.

*ec2ctl -reboot -filter BOB*

Reboot all Bob's instances. If there are more than 2, add *-force*.

*ec2ctl -terminate i-ab64efaa*

Terminate the instance with the specified instance ID. 

