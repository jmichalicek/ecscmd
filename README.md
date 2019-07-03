# ecscmd
A simple utility for working with Amazon ECS

The initial functionality is aimed specifically at updating task definitions and services using the Fargate launch type.

## Why?
ECS deployments can fall into a strange middleground where the best tool for the job appears to be infrastructure management tools such as terraform. Tools like that can get clunky for deployments for a few reasons. Sometimes they manage state weird ways such as new task definition versions in Terraform. Due to maintaining the state and relations of everything involved the potentially affected surface area can be very large and so updating true infrastructure can become difficult and need temporary one off changes to things like Terraform code to ignore certain changes. Similarly just updating a task definition or service to use a new task definition has the risk of touching more things than is needed.

This tool is aimed at moving the task definition and service updates out of tools like Terraform to make the state management simpler and reduce the surface area for what might be affected. This also simplifies deployments for developers on the team who do not necessarily need or want to also manage tools such as Terraform or CloudFormation and other infrastructure state. The separation of concerns can also make it easier for organizations with a need for tighter controls over who can access or modify different infrastructure resources to more easily allow developers to deploy to ECS without giving unnecessary access to other things.
