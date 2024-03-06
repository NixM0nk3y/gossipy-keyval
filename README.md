# Welcome to Gossipy Key Value

Foo

## Architecture

![diagram](_media/Architecture.png ":size=25%")

## Setup

`aws ecs put-account-setting-default --name dualStackIPv6 --value enabled --region eu-west-1`

## Useful commands

-   `make clean` remove any intermediate state
-   `make diff` compare deployed stack with current state
-   `make deploy ` deploy this stack to your default AWS account/region
