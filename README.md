# From code to Openshift

### Prereq's

A computer with `Minishift`, `git` & `oc` installed


### Tasks

1) Build a docker container from the Dockerfile, run the container, make a curl request (expose ports).

2) Save the image to a tar.gz (`docker export <container> -o example.tar.gz`)

3) Start minishift, create a new project named: `example`

4) Import the saved docker image from step 3) above `docker import example.tar.gz $(minishift openshift registry)/example/example:latest`. (`eval $(minishift docker-env`)

5) Push the Docker image from step 4 as `example:latest` to the `example` namespace. (`eval $(minishift docker-env) && docker login -u developer -p $(oc whoami -t) $(minishift openshift registry)`)

6) Apply the `deployment.yaml` file into the `example` project.

7) Run curl against the route from your desktop

8) Run `hey` from a shell in the pod running in minishift

