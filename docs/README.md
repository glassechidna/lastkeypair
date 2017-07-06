# `lastkeypair`

[![Build Status](https://travis-ci.org/glassechidna/lastkeypair.svg?branch=master)](https://travis-ci.org/glassechidna/lastkeypair)

**NOTE: README is a work-in-progress**

## Preamble

`lastkeypair` was borne out of a frustration with the proliferation of SSH
key-pairs that is all too common across large AWS deployments. These are the 
normal sort of pains: 

* Developers freely create new keys because they don't have access to existing 
  keys. These are only stored on their dev laptops and maybe shared with others
  on an ad-hoc basis.
* Golden keys are centrally administered by a benevolent ops teams that does a 
  decent job of key-rotation, key-sharing and so on. Keys need to be immediately 
  replaced when an employee leaves.
* Every developer has their own key and all keys are distributed to all 
  instances - maybe as a boot job and on an hourly basis thereafter.
* People decide all the above options are awful and resort to some kind of LDAP
  module wherein `sshd` on instances makes a call back to a central login 
  server - better hope that server is up and there is connectivity at 2AM when
  a system is on fire.
  
OpenSSH has a long-supported but woefully under-utilised certificate 
functionality. This is conceptually _similar_ to X509 (i.e. "SSL certs"), but
SSH-specific. `lastkeypair` aims to be a plug-and-play solution based on SSH 
certificates, AWS Lambda and AWS KMS.

## Setup

**NOTE: Add first-time admin setup**

Once your administrator has setup `lastkeypair` (LKP) there are a couple of 
things you do differently. Your instance initialisation has an addition step 
and you SSH into machines with a new command.

### EC2 instance setup

On the EC2 side of things, you set up your instances to trust the LKP SSH
certificate authority. You do this by choosing the LKP SSH keypair when starting
your instance. Secondly, in your userdata script (or whichever instance 
initialisation system you use) you run the LKP binary. This binary retrieves the
instance ID and its SSH host key and sends both of these to the LKP Lambda
function. The Lambda returns a _signed_ host certificate and the LKP binary
configures the SSH server to:

* Present the signed host certificate to users. This prevents the "unknown 
  host key" prompt when connecting to a server for the first time.
* Trust the LKP SSH CA for when users log in.
* Create a list of "authorised principals" that ensures can only log in
  when they've explicitly told the LKP Lambda the exact instance they want
  to SSH into. This prevents one user cert from being valid for _any_ one
  of your instances (i.e. authentication without authorisation).
  
To do this you add this to your userdata:

    curl -L -o lkp https://github.com/glassechidna/lastkeypair/releases/download/0.0.2/lastkeypair_linux_amd64
    chmod +x lkp
    ./lkp host # there are several flags you can pass in here if your Lambda or KMS key alias aren't the default
    service sshd restart # This is correct for Amazon Linux, can be different on other distros
    rm lkp
    
### User laptop setup

The setup on your laptop is simpler than configuring your EC2 instances. 

Firstly download the correct LKP binary for your operating system from the 
[GitHub Releases][https://github.com/glassechidna/lastkeypair/releases] page
and place it somewhere on your `PATH`. 

If your administrator selected the default LKP Lambda and KMS key alias, you can
now simply type:
  
    $ lkp ssh exec --instance-arn arn:aws:ec2:us-east-1:9876543210:instance/i-0123abcd -- ec2-user@<ip>
    
This fetches a time-limited SSH certificate valid for the selected instance and 
initiates an SSH connection to it. Any flags passed after `--` are passed directly
to the underlying `ssh` invocation.

## How it works

![lastkeypair-sequence-diagram](sequence-diagram.png)

## Alternatives

* [BLESS](https://github.com/netflix/bless)
* [python-blessclient](https://github.com/lyft/python-blessclient)
* [sshephalopod](https://github.com/realestate-com-au/sshephalopod/)
* [ssh-cert-authority](https://github.com/cloudtools/ssh-cert-authority)
* [pam-ussh](https://github.com/uber/pam-ussh)
* [facebook-doc](https://code.facebook.com/posts/365787980419535/scalable-and-secure-access-with-ssh/)

