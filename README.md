# ecscmd
A simple utility for working with Amazon ECS

The initial functionality is aimed specifically at updating task definitions and services using the Fargate launch type.

## Why?
ECS deployments can fall into a strange middleground where the best tool for the job appears to be infrastructure management tools such as terraform. Tools like that can get clunky for deployments for a few reasons. Sometimes they manage state weird ways such as new task definition versions in Terraform. Due to maintaining the state and relations of everything involved the potentially affected surface area can be very large and so updating true infrastructure can become difficult and need temporary one off changes to things like Terraform code to ignore certain changes. Similarly just updating a task definition or service to use a new task definition has the risk of touching more things than is needed.

This tool is aimed at moving the task definition and service updates out of tools like Terraform to make the state management simpler and reduce the surface area for what might be affected. This also simplifies deployments for developers on the team who do not necessarily need or want to also manage tools such as Terraform or CloudFormation and other infrastructure state. The separation of concerns can also make it easier for organizations with a need for tighter controls over who can access or modify different infrastructure resources to more easily allow developers to deploy to ECS without giving unnecessary access to other things.

## Usage
ecscmd register-task-def <name> [--config=~/.ecscmd.toml]

## AWS Config
* Ensure task execution role is set up - https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_execution_IAM_role.html
* registering a new task def requires more settings configured than just updating an existing task def.


### Other
https://github.com/fabfuel/ecs-deploy was the original inspiration for this, I believe, but I was unable to actually find it when I wanted to use it. It currently does not quite do what I want and I wanted to avoid users needing to muck with Python related stuff.

## TODO:
* Try to load config from current directory first THEN home to allow for project specific configs.
* Consider loading template vars at top of namespace - right now they have an ugly nested `taskDefinition.name.templateVars.VarName` for the full environment variable. Gross. Perhaps the in config nested ones can override a top level defaults.
* Namespace environment vars used for template vars so no accidents happen?  `ECSCMD_DjangoSettingsModule`? If so, keep literal or do some automatic conversion from camel cased to upper snake case?
* run tasks
* update service
