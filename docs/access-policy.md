# LastKeypair Access Control Policies

Sometimes you don't want to grant all authenticated users access to all instances 
in your AWS account. You might work on isolated teams, you might grant third 
parties access to selected machines or your company might require at least two 
people to sign off on access to particularly sensitive machines. Maybe you just 
want to mandate a work-life balance and disable SSH access for people not 
on-call after 5PM.

LastKeypair supports these use-cases with access policies. Rather than try to 
shoehorn this into some IAM policy monstrosity - or worse yet, make you learn
another cryptic JSON schema - we have opted to make the language standard
Javascript.

## Usage

The Javascript environment is provided by [Duktape](http://duktape.org/). LKP
will execute a `validate()` function with a `context` object and expect an object
response.

Here we will use Typescript notation to describe the context object, response object
and functions available to you in the JS environment.

```typescript
interface LkpContext {
    fromName?: string; // IAM username - not set for assumed roles (e.g. SAML users)
    fromId: string; // IAM (role/user) unique ID 
    fromAccount: string; // AWS numeric account ID containing user
    to: "LastKeypair";
    type: "User" | "AssumedRole" | "FederatedUser"; // type of user in 'from' fields
    remoteInstanceArn: string; // instance ARN that user is requesting access to
    
    voucherAccount?: string; // in two-person authorisations, these fields mirror 
    voucherId?: string;      // the 'from' fields, albeit for the user doing the "vouching" 
    voucherName?: string;
    voucherInstanceArn?: string;
}

interface LkpValidateResponse {
    authorized: boolean;
}

// ec2tags() returns a string->string map of all ec2 tags for a given instance ARN
interface KeyValMap {
    [key: string]: string;
}
function ec2tags(remoteInstanceArn: string): KeyValMap;

// userGroups() returns an array of group names that the given user belongs to
function userGroups(awsAccountId: string, iamUsername: string): [string];
```

## Example

This is a somewhat exhaustive example of the sorts of policies you might enact.

```javascript
function validate(context) {
    // we allow multiple a third-party account ssh access and we don't want them to be
    // sneaky and create IAM users with the same name as us. we _could_ use IAM unique IDs
    // but in this case we'd prefer to check (acctID, username) tuples.
    var isMainAccount = context.fromAccount === '9876543210';
    
    var now = new Date();
    var hour = now.getUTCHours();
    var authorized = { authorized: true };

    if (isMainAccount && context.fromName === 'aidan.steele@glassechidna.com.au') return authorized; // aidan is all powerful

    if (isMainAccount && context.fromName === 'benjamin.dobell@glassechidna.com.au') {
        if (hour >= 9 && hour < 17) return authorized; // ben usually only has access during work hours
    }
    
    if (context.fromAccount === '01234567890') { // aws account id of 3rd-party support provider
        if (hour < 9 || hour >= 17) return authorized; // our trusted partner is allowed in outside of work hours
    }

    var rolePrefix = "AROAIIWP2XR7EN6EXAMPLE:";
    if (isMainAccount && context.fromId.indexOf(rolePrefix) === 0) {
        var roleSessionName = context.fromId.substr(rolePrefix.length);
        // dan isn't an IAM user (he uses SAML to log into AWS) so we check the role session
        // name from his sts:AssumeRole call
        if (roleSessionName === 'daniel.whyte@glassechidna.com') return authorized;
    }

    var partyHost = 'arn:aws:ec2:ap-southeast-2:9876543210:instance/i-0123abcd';
    if (context.remoteInstanceArn === partyHost) return authorized; // we'll let anyone on our party box

    var devRegion = 'arn:aws:ec2:us-east-1:9876543210';
    if (context.remoteInstanceArn.indexOf(devRegion) === 0) return authorized; // the dev region is a free-for-all

    var uatRegion = 'arn:aws:ec2:us-east-2:9876543210';
    if (context.remoteInstanceArn.indexOf(uatRegion) === 0) {
        var groups = userGroups(context.fromAccount, context.fromId);
        if (groups.indexOf("Developers") >= 0) return authorized; // the uat region is only open to devs
    }

    if (ec2tags(context.remoteInstanceArn).clearanceLevel === 'super-secure') {
        // some of our instances have a clearanceLevel=super-secure tag on them. only ben is allowed
        // to log into these machines, but only if aidan has vouched for him (sent him an approval
        // token over slack, email, etc)
        if (
            isMainAccount &&
            context.fromName === 'benjamin.dobell@glassechidna.com.au' &&
            context.voucherAccount === '9876543210' &&
            context.voucherName === 'aidan.steele@glassechidna.com.au' &&
            context.voucherInstanceArn === context.remoteInstanceArn // aidan only vouched for _this_ machine, not all super-secure machines!
        ) return authorized;
    }

    return { authorized: false };
}
```
