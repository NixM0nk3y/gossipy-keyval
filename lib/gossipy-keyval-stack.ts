import * as cdk from "aws-cdk-lib";
import { Construct } from "constructs";
import { GossipyKeyVal } from "./constructs/gossipykeyval";
import { IPv6Network } from "./constructs/ipv6vpc";
import { Cluster } from "aws-cdk-lib/aws-ecs";
import { PrivateDnsNamespace } from "aws-cdk-lib/aws-servicediscovery";

export interface GossipyKeyProps extends cdk.StackProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
}

export class GossipyKeyValStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: GossipyKeyProps) {
        super(scope, id, props);

        const network = new IPv6Network(this, "IPv6Network", {
            tenant: props.tenant,
            environment: props.environment,
            product: props.product,
        });

        const cluster = new Cluster(this, "Cluster", { vpc: network.vpc });

        const namespace = new PrivateDnsNamespace(this, "CloudMapNamespace", {
            vpc: network.vpc,
            name: "domain.local",
        });

        new GossipyKeyVal(this, "GossipyKeyVal", {
            tenant: props.tenant,
            environment: props.environment,
            product: props.product,
            cluster: cluster,
            namespace: namespace,
            vpc: network.vpc,
        });
    }
}
