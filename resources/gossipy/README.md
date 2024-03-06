# Welcome to GossipyKV

Test Bed Project to demonstrate a cluster being stood up in ECS fargate that coordinates actions across the cluster with a gossip clustering algorithm.

In this case the app in a very rough distributed eventually consistent keystore API.

## Architecture

Each of the three tasks of the cluster hosts a rest API supporting basics operations to PUT/GET/DELETE values from the the tasks local keystore ( in-memory).

REST Operations are propagated to the remaining member of the cluster using broadcast messages over a gossip protocol. Cluster members receiving the messages will replay those actions on their own keystores.

![diagram](_media/Gossipy.png ":size=25%")

```
$ # add a value into the keystore
$curl -X PUT http://GossipyKeyValLB-1018863412.eu-west-1.elb.amazonaws.com/kv/foo/bar
{"foo":"bar"}

$ # retrieve a value from the keystore
$curl http://GossipyKeyValLB-1018863412.eu-west-1.elb.amazonaws.com/kv/foo
{"foo":"bar"}

$ # delete a value from the keystore
$curl -X DELETE http://GossipyKeyValLB-1018863412.eu-west-1.elb.amazonaws.com/kv/foo

$ # confirm the deletion
$ curl -v http://GossipyKeyValLB-1018863412.eu-west-1.elb.amazonaws.com/kv/foo
*   Trying 54.72.83.9:80...
* Connected to GossipyKeyValLB-1018863412.eu-west-1.elb.amazonaws.com (54.72.83.9) port 80 (#0)
> GET /kv/foo HTTP/1.1
> Host: GossipyKeyValLB-1018863412.eu-west-1.elb.amazonaws.com
> User-Agent: curl/7.86.0
> Accept: */*
>
* Mark bundle as not supporting multiuse
< HTTP/1.1 404 Not Found
< Date: Wed, 06 Mar 2024 2
```

## Useful commands

-   `make clean` remove any intermediate state
-   `make diff` compare deployed stack with current state
-   `make deploy ` deploy this stack to your default AWS account/region
