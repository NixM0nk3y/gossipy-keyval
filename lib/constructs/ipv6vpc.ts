import { Construct } from "constructs";
import { CfnOutput } from "aws-cdk-lib";
import {
    Vpc,
    VpcProps,
    CfnVPCCidrBlock,
    CfnEgressOnlyInternetGateway,
    RouterType,
    CfnSubnet,
    PrivateSubnet,
    PublicSubnet,
    SubnetType,
    NatProvider,
} from "aws-cdk-lib/aws-ec2";
import { Fn } from "aws-cdk-lib";

export interface IPv6VpcProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
}

export class IPv6Network extends Construct {
    public vpc: Vpc;

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    constructor(scope: Construct, id: string, props: IPv6VpcProps) {
        super(scope, id);

        // define our nat provider outside the VPC - we need it later
        const natGatewayProvider = NatProvider.gateway();

        // fire up our IPv6 Enabled Vpc
        this.vpc = new IPv6Vpc(this, "ElasticIPv6VPC", {
            maxAzs: 3,
            natGatewayProvider,
            natGateways: 1,
            enableDnsSupport: true,
            enableDnsHostnames: true,
            subnetConfiguration: [
                {
                    subnetType: SubnetType.PUBLIC,
                    name: "public",
                    cidrMask: 24,
                },
                {
                    subnetType: SubnetType.PRIVATE_WITH_EGRESS,
                    name: "private",
                    cidrMask: 24,
                },
            ],
        });

        // extract our nat gateway ( we assume just one )
        const natgatewayId = natGatewayProvider.configuredGateways[0].gatewayId;

        // attach our NAT64 route
        this.vpc.privateSubnets.forEach((subnet) => {
            const s = subnet as PrivateSubnet;
            s.addRoute("ipv6Nat64Route", {
                routerType: RouterType.NAT_GATEWAY,
                routerId: natgatewayId,
                destinationIpv6CidrBlock: "64:ff9b::/96",
            });
        });

        new CfnOutput(this, "IPv6Network", {
            value: this.vpc.vpcId,
            description: "VPCId",
        });
    }
}

class IPv6Vpc extends Vpc {
    constructor(scope: Construct, id: string, props?: VpcProps) {
        super(scope, id, props);

        const ip6cidr = new CfnVPCCidrBlock(this, "Cidr6", {
            vpcId: this.vpcId,
            amazonProvidedIpv6CidrBlock: true,
        });

        const vpc6cidr = Fn.select(0, this.vpcIpv6CidrBlocks);
        const subnet6cidrs = Fn.cidr(vpc6cidr, 256, (128 - 64).toString());

        const allSubnets = [...this.publicSubnets, ...this.privateSubnets, ...this.isolatedSubnets];

        allSubnets.forEach((subnet, i) => {
            const cidr6 = Fn.select(i, subnet6cidrs);
            const cfnSubnet = subnet.node.defaultChild as CfnSubnet;
            cfnSubnet.ipv6CidrBlock = cidr6;
            cfnSubnet.assignIpv6AddressOnCreation = true;
            cfnSubnet.enableDns64 = true;
            subnet.node.addDependency(ip6cidr);
        });

        if (this.publicSubnets) {
            this.publicSubnets.forEach((subnet) => {
                const s = subnet as PublicSubnet;
                s.addRoute("DefaultRoute6", {
                    routerType: RouterType.GATEWAY,
                    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                    routerId: this.internetGatewayId!,
                    destinationIpv6CidrBlock: "::/0",
                    enablesInternetConnectivity: true,
                });
            });
        }

        if (this.privateSubnets) {
            const eigw = new CfnEgressOnlyInternetGateway(this, "EgressOnly", {
                vpcId: this.vpcId,
            });

            this.privateSubnets.forEach((subnet) => {
                const s = subnet as PrivateSubnet;
                s.addRoute("DefaultRoute6", {
                    routerType: RouterType.EGRESS_ONLY_INTERNET_GATEWAY,
                    routerId: eigw.ref,
                    destinationIpv6CidrBlock: "::/0",
                    enablesInternetConnectivity: true,
                });
            });
        }
    }
}
