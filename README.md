lkp
  token
    create      # creates a token that can be authenticated by kms
    validate    # validates aforementioned token 
  ssh
    sign        # uses CA key to sign user or host key
    exec        # ask lambda func to sign ssh pubkey
    proxy       # to be used as ssh_config ProxyCommand. maybe allows user@i-<instance>?
  setup         # creates kms key+policy, ssh CA key, uploads lambda zip, everything
    --dry-run   # just emits cfn files, zip, ssh key, etc
    --do-it     # actually performs all the actions
  ec2
    sign        # sends host key to lambda, replaces instance key with signed version
    trustca     # adds 'cert-authority' flag to ~/.ssh/authorized_keys entry
  vouch         # create token to send out-of-bound to person who needs 2-operator login
    --recipient
    --duration
    --host
  lambda        # fulfils the lambda func, is passed fn args in stdin by thin wrapper
