import { Construct } from "constructs";
import { CfnOutput, Duration, RemovalPolicy } from "aws-cdk-lib";
import {
    ContainerImage,
    Cluster,
    OperatingSystemFamily,
    CpuArchitecture,
    CfnService,
    FargateTaskDefinition,
    LinuxParameters,
    LogDrivers,
    Protocol,
} from "aws-cdk-lib/aws-ecs";
import { Vpc, Port, SecurityGroup } from "aws-cdk-lib/aws-ec2";
import { PolicyStatement, Effect } from "aws-cdk-lib/aws-iam";
import { LogGroup, RetentionDays } from "aws-cdk-lib/aws-logs";
import { DnsRecordType, PrivateDnsNamespace } from "aws-cdk-lib/aws-servicediscovery";
import { ApplicationLoadBalancedFargateService } from "aws-cdk-lib/aws-ecs-patterns";

export interface GossipyKeyValProps {
    readonly tenant: string;
    readonly environment: string;
    readonly product: string;
    readonly cluster: Cluster;
    readonly namespace: PrivateDnsNamespace;
    readonly vpc: Vpc;
}

export class GossipyKeyVal extends Construct {
    constructor(scope: Construct, id: string, props: GossipyKeyValProps) {
        super(scope, id);

        // basic ecs exec role perms
        const executionRolePolicy = new PolicyStatement({
            effect: Effect.ALLOW,
            resources: ["*"],
            actions: [
                "ecr:GetAuthorizationToken",
                "ecr:BatchCheckLayerAvailability",
                "ecr:GetDownloadUrlForLayer",
                "ecr:BatchGetImage",
                "logs:CreateLogStream",
                "logs:PutLogEvents",
            ],
        });

        // setup our task def basics
        const taskDefinition = new FargateTaskDefinition(this, "TaskDef", {
            cpu: 256,
            memoryLimitMiB: 512,
            runtimePlatform: {
                operatingSystemFamily: OperatingSystemFamily.LINUX,
                cpuArchitecture: CpuArchitecture.ARM64,
            },
        });

        taskDefinition.addToExecutionRolePolicy(executionRolePolicy);

        // ssm permissions
        taskDefinition.addToTaskRolePolicy(
            new PolicyStatement({
                resources: ["*"],
                actions: [
                    "ssmmessages:CreateControlChannel",
                    "ssmmessages:CreateDataChannel",
                    "ssmmessages:OpenControlChannel",
                    "ssmmessages:OpenDataChannel",
                ],
                effect: Effect.ALLOW,
            }),
        );

        const logGroup = new LogGroup(this, "LogGroup", {
            logGroupName: `/${props.tenant.toLowerCase()}/${props.product.toLowerCase()}/${props.environment.toLowerCase()}/ecs`,
            retention: RetentionDays.ONE_WEEK,
            removalPolicy: RemovalPolicy.DESTROY,
        });

        const container = taskDefinition.addContainer("Caddy", {
            image: ContainerImage.fromAsset("./resources/gossipy"),
            logging: LogDrivers.awsLogs({ streamPrefix: "GossipKV", logGroup: logGroup }),
            containerName: "gossipkv",
            linuxParameters: new LinuxParameters(this, "LinuxParams", {
                initProcessEnabled: true,
            }),
            healthCheck: {
                command: ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
                interval: Duration.seconds(60),
                retries: 3,
                startPeriod: Duration.seconds(60),
                timeout: Duration.seconds(5),
            },
            environment: {
                LOG_LEVEL: "INFO",
                SERVICE_DISCOVERY_HOST: `gossipkeyval.${props.namespace.privateDnsNamespaceName}`,
            },
        });

        // API port
        container.addPortMappings({
            containerPort: 8080,
            hostPort: 8080,
            protocol: Protocol.TCP,
        });

        // cluster ports
        container.addPortMappings({
            containerPort: 7947,
            hostPort: 7947,
            protocol: Protocol.TCP,
        });
        container.addPortMappings({
            containerPort: 7947,
            hostPort: 7947,
            protocol: Protocol.UDP,
        });

        // our container security group
        const serviceSG = new SecurityGroup(this, "SecurityGroup", {
            vpc: props.vpc,
            allowAllOutbound: true,
            allowAllIpv6Outbound: true,
            description: "Security group for the service",
        });

        serviceSG.node.addDependency(props.vpc);

        // allow intra-cluster traffic
        serviceSG.addIngressRule(serviceSG, Port.tcp(7947), "allow intra cluster traffic");
        serviceSG.addIngressRule(serviceSG, Port.udp(7947), "allow intra cluster traffic");

        // our service cluster
        const loadBalancedFargateService = new ApplicationLoadBalancedFargateService(this, "Service", {
            cluster: props.cluster,
            circuitBreaker: {
                rollback: true,
            },
            desiredCount: 3,
            publicLoadBalancer: true,
            securityGroups: [serviceSG],
            capacityProviderStrategies: [
                {
                    capacityProvider: "FARGATE",
                    weight: 1,
                },
            ],
            cloudMapOptions: {
                name: "gossipkeyval",
                cloudMapNamespace: props.namespace,
                dnsRecordType: DnsRecordType.SRV,
            },
            taskDefinition: taskDefinition,
            taskSubnets: {
                subnets: props.vpc.privateSubnets,
            },
            loadBalancerName: "GossipyKeyValLB",
        });

        // speed up cluster deploys
        loadBalancedFargateService.targetGroup.setAttribute("deregistration_delay.timeout_seconds", "10");

        loadBalancedFargateService.targetGroup.configureHealthCheck({
            path: "/health",
            interval: Duration.seconds(30),
            healthyThresholdCount: 3,
            unhealthyThresholdCount: 6,
        });

        // allow ecs exec to cluster
        const cfnService = loadBalancedFargateService.service.node.defaultChild as CfnService;
        cfnService.enableExecuteCommand = true;

        new CfnOutput(this, "ApiURI", {
            value: `${loadBalancedFargateService.loadBalancer.loadBalancerDnsName}`,
            description: "The API Endpoint",
        });
    }
}
