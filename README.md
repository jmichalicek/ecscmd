# ecscmd
A simple utility for working with Amazon ECS

The initial functionality is aimed specifically at updating task definitions and services using the Fargate launch type.

## Why?
ECS deployments can fall into a strange middleground where the best tool for the job appears to be infrastructure management tools such as terraform. Tools like that can get clunky for deployments for a few reasons. Sometimes they manage state weird ways such as new task definition versions in Terraform. Due to maintaining the state and relations of everything involved the potentially affected surface area can be very large and so updating true infrastructure can become difficult and need temporary one off changes to things like Terraform code to ignore certain changes. Similarly just updating a task definition or service to use a new task definition has the risk of touching more things than is needed.

This tool is aimed at moving the task definition and service updates out of tools like Terraform to make the state management simpler and reduce the surface area for what might be affected. This also simplifies deployments for developers on the team who do not necessarily need or want to also manage tools such as Terraform or CloudFormation and other infrastructure state. The separation of concerns can also make it easier for organizations with a need for tighter controls over who can access or modify different infrastructure resources to more easily allow developers to deploy to ECS without giving unnecessary access to other things.

There are already a few tools aimed at this, but none quite fit my primary requirements:
* No external dependencies
* Updating task definitions independently of updating a service - some task definitions only exist to run as one off tasks.
* Support for more complex task definitions which may have multiple containers running different images
* Ability run one off ecs tasks

## Usage
The first `<name>` is the name of the section in your config file. Not all flags are yet documented here, just examples. The config section name may become optional and possibly a flag using `--some-flag` so that you can run without having a section in your config.
```
ecscmd task register <name> [--config=~/.ecscmd.toml]
ecscmd task run <name> [--stream-output --config=~/.ecscmd.toml]
ecscmd service update <name> [--task-definition=mytaskdefinition:1 --force-deployment --config=~/.ecscmd.toml]
ecscmd service create <name> [--name=myservice --task-definition=mytaskdefinition:1 --force-deployment --config=~/.ecscmd.toml]
```

### Docker

```
docker pull jmichalicek/ecscmd:alpine`
docker run -v .:/ecscmd jmichalicek/ecscmd:alpine task update mytaskdef --config /ecscmd/.ecscmd.yaml`
```

## AWS Config
* Ensure task execution role is set up - https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_execution_IAM_role.html
* registering a new task def requires more settings configured than just updating an existing task def.


### Other
* https://github.com/fabfuel/ecs-deploy was the original inspiration for this, I believe, but I was unable to actually find it when I wanted to use it. It currently does not quite do what I want and I wanted to avoid users needing to muck with Python related stuff.
* https://github.com/nathanpeck/awesome-ecs Amazon's Nathan Peck also maintains a list of ECS tools, including things similar to this which may fit your specific needs better

## TODO:
* run tasks - wait for task to complete, etc.
* Nicer, easier to read output + json output like now, but not via logger. Possibly a couple levels of output or way to configure what values to output for easy piping into other commands
* Include container name in cloudwatch log streaming output
* Tons of internal API cleanup - decisions need to be made about how directly to just use the aws sdk vs wrap it and hide it from the outer layers (the latter would result in a lot of mirroring existing aws stuff), passing around lots of map[string]interface{} currently which could be typed structs, and moving cobra and koanf init code OUTSIDE of init() functions.
* Pass properly typed data instead of map[string]interface{} created from Koanf around
* Look up task def from AWS and dump the json for containers as a template
* Consider going to just entire task def as json template?
* `--dry-run` flag which just shows the json request which would be sent to AWS for register task def and service create/update
* Look up current service and use as base for configs to reduce config params which must be kept in sync locally with what is desired on remote. Do same for task defs if possible.
* Support multiple configs `--config=file1.yaml --config=file2.yaml` where later configs override/add to earlier parsed configs
* Support `default` section in config files
* Support CODE_DEPLOY controller and external controller/task sets for ECS services
* more command line params supported for register task definition - particularly template variables
* support passing aws auth info and aws profile on command line
* Flatten configs/names. Instead of `taskDefinition.name` and `service.name`, just `name` which could be `service-name` or `account-service-name`or `taskdef-foo`, etc. however users need which will simplify and flatten configs.
 a top level defaults. <- currently not likely to do this
* Namespace environment vars used for template vars so no accidents happen?  `ECSCMD_DjangoSettingsModule`? If so, keep literal or do some automatic conversion from camel cased to upper snake case?
* Clean up code for config and command line parsing
* Replace Cobra with something which allows for nil/unset values from the command line to better distinguish between "not specified" and "specified as the zero value" and provide more accurate generated help.
