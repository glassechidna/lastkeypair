# LastKeypair Access Control Policies

Sometimes you don't want to grant all authenticated users access to all instances 
in your AWS account. You might work on isolated teams, you might grant third 
parties access to selected machines or your company might require at least two 
people to sign off on access to particularly sensitive machines. Maybe you just 
want to mandate a work-life balance and disable SSH access for people not 
on-call after 5PM.

LastKeypair supports these use-cases with access policies. Rather than try to 
shoehorn this into some IAM policy monstrosity - or worse yet, make you learn
another cryptic rules-engine JSON schema - we have opted to factor out authorisation
into a separate Lambda function.

If you specify an `AUTHORIZATION_LAMBDA` environment variable, LKP will execute
that Lambda function in order to determine if a user is authorised to SSH into
their requested instance. You are free to structure your Lambda function however
you please. 

The format of the Lambda's request parameters and the expected response are 
documented in Typescript notation. 

```typescript
interface LkpAuthorizationRequest{
    FromName?: string; // IAM username - not set for assumed roles (e.g. SAML users)
    FromId: string; // IAM (role/user) unique ID 
    FromAccount: string; // AWS numeric account ID containing user
    Type: "User" | "AssumedRole" | "FederatedUser"; // type of user in 'from' fields
    RemoteInstanceArn: string; // instance ARN that user is requesting access to
    
    VoucherAccount?: string; // in two-person authorisations, these fields mirror 
    VoucherId?: string;      // the 'from' fields, albeit for the user doing the "vouching" 
    VoucherName?: string;11
    VoucherInstanceArn?: string;
}

interface LkpAuthorizationResponse {
    Authorized: boolean;
    Jumpbox?: { 
        IpAddress: string; // ip that user should use as bastion host
        InstanceId: string; // LKP uses instance ids as principals for trusted hosts
        User: string; // linux user on jumpbox
    };
    CertificateOptions?: { // as per https://man.openbsd.org/ssh-keygen#O
        ForceCommand?: string;
        SourceAddress?: string;
    };
}
```

## Example

This is a somewhat exhaustive example of the sorts of policies you might enact.

```javascript
exports.handler = function(event, context, callback) {
    // we allow multiple a third-party account ssh access and we don't want them to be
    // sneaky and create IAM users with the same name as us. we _could_ use IAM unique IDs
    // but in this case we'd prefer to check (acctID, username) tuples.
    var isMainAccount = event.fromAccount === '9876543210';
    
    var now = new Date();
    var hour = now.getUTCHours();
    var authorized = function() { callback({ authorized: true }) };

    if (isMainAccount && event.fromName === 'aidan.steele@glassechidna.com.au') authorized(); // aidan is all powerful

    if (isMainAccount && event.fromName === 'benjamin.dobell@glassechidna.com.au') {
        if (hour >= 9 && hour < 17) authorized(); // ben usually only has access during work hours
    }
    
    if (event.fromAccount === '01234567890') { // aws account id of 3rd-party support provider
        if (hour < 9 || hour >= 17) authorized(); // our trusted partner is allowed in outside of work hours
    }

    var rolePrefix = "AROAIIWP2XR7EN6EXAMPLE:";
    if (isMainAccount && event.fromId.indexOf(rolePrefix) === 0) {
        var roleSessionName = event.fromId.substr(rolePrefix.length);
        // dan isn't an IAM user (he uses SAML to log into AWS) so we check the role session
        // name from his sts:AssumeRole call
        if (roleSessionName === 'daniel.whyte@glassechidna.com') authorized();
    }

    var partyHost = 'arn:aws:ec2:ap-southeast-2:9876543210:instance/i-0123abcd';
    if (event.remoteInstanceArn === partyHost) authorized(); // we'll let anyone on our party box

    var devRegion = 'arn:aws:ec2:us-east-1:9876543210';
    if (event.remoteInstanceArn.indexOf(devRegion) === 0) authorized(); // the dev region is a free-for-all

    callback({ authorized: false });
}
```
