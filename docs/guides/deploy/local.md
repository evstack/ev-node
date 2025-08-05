# 🏠 Local

This tutorial is going to show you how to deploy the [gm-world chain](/guides/gm-world.md) using Docker Compose.

You can learn more about Docker Compose [here](https://docs.docker.com/compose/).

<!-- markdownlint-disable MD033 -->
<script setup>
import Callout from '../../.vitepress/components/callout.vue'
import constants from '../../.vitepress/constants/constants.js'
</script>

:::tip
<Callout />
:::
<!-- markdownlint-enable MD033 -->

## 💻 Pre-requisites {#prerequisites}

Make sure you have your gm-world chain ready by completing [the Build your chain tutorial](/guides/gm-world.md).

## 🛠️ Dependencies {#dependencies}

### 💻 Docker Compose {#docker-compose}

You can [install docker compose here](https://docs.docker.com/compose/install/).

Once installed, you can verify the installation by running:

```bash
docker compose version
```

```bash
Docker Compose version v2.23.0-desktop.1
```

## 🛠️ Setting up your environment {#setting-up-your-environment}

In addition to our chain, we need to run a DA.

We will use the [local-da](https://github.com/evstack/ev-node/tree/main/da/cmd/local-da) for this tutorial and run it with our chain.

To save time, we can use the local-da Dockerfile:

* [local-da Dockerfile](https://github.com/evstack/ev-node/blob/main/Dockerfile.da)
This will allow us to focus on how we can run the gm-world chain with Docker Compose.

### 🐳 Dockerfile {#dockerfile}

First, we need to create a Dockerfile for our gm-world chain. Create a new file called `Dockerfile.gm` in the root of the `gm` directory and add the following code:

```dockerfile-vue
# Stage 1: Install ignite CLI and evolve
FROM golang AS base

# Install dependencies
RUN apt update && \
 apt-get install -y \
 build-essential \
 ca-certificates \
 curl

RUN curl -sSL https://evolve.dev/install.sh | bash
# Install evolve

# Install ignite
RUN curl https://get.ignite.com/cli! | bash

# Set the working directory
WORKDIR /app

# cache dependencies.
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download

# Copy all files from the current directory to the container
COPY . .

# Build the chain
RUN ignite app install -g github.com/ignite/apps/evolve
RUN ignite chain build  -y
RUN ignite evolve init

# Stage 2: Set up the runtime environment
FROM debian:bookworm-slim

# Install jq
RUN apt update && \
 apt-get install -y \
 jq

# Set the working directory
WORKDIR /root

# Copy over the evolve binary from the build stage
COPY --from=base /go/bin/gmd /usr/bin


# Copy the $HOME/.gm directory from the build stage.
# This directory contains all your chain config.
COPY --from=base /root/.gm /root/.gm

# Keep the container running after it has been started
# CMD tail -f /dev/null

ENTRYPOINT ["gmd"]
CMD ["start","--evolve.node.aggregator"]
```

This Dockerfile sets up the environment to build the chain and run the gm-world node. It then sets up the runtime environment to run the chain. This allows you as the developer to modify any files, and then simply rebuild the Docker image to run the new chain.

Build the docker image by running the following command:

```bash
docker build -t gm-world -f Dockerfile.gm .
```

```bash
cd evolve
docker build -t local-da -f Dockerfile.da .
cd ..
```

You can then see the built image by running:

```bash
docker images
```

You should see the following output:

```bash
REPOSITORY  TAG     IMAGE ID       CREATED         SIZE
gm-world    latest  5d3533c1ea1c   8 seconds ago   443MB
```

### 🐳 Docker Compose file {#docker-compose-file}

Next we need to create our `compose.yaml` file for docker compose to use.

In the root of the `gm` directory, create a new file called `compose.yaml` and add the following code:

```yml-vue
services:
  # Define the gm-world chain service
  gm-world:
    # Set the name of the docker container for ease of use
    container_name: gm-world
    # Use the image we just built
    image: gm-world
    # Used for networking between the two services
    network_mode: host
    # The command config is used for launching the chain once the Docker container is running
    command:
      [
        "start",
        "--evolve.node.aggregator",
        "--evolve.da.address",
        "http://0.0.0.0:7980",
      ]
    # Ensures the local-da service is up and running before starting the chain
    depends_on:
      - local-da

  # Define the local DA service
  local-da:
    # Use the published image from evolve
    image: local-da
      # Set the name of the docker container for ease of use
    container_name: local-da
    # Publish the ports to connect
    ports:
      - "7980:7980"

```

We now have all we need to run the gm-world chain and connect to a local DA node.

### 🚀 Run gm-world chain {#run-gm-world-chain}

Run your gm-world chain by running the following command:

```bash
docker compose up
```

You'll see logs of your chain being output.

Congratulations! You have successfully run the gm-world chain with Docker Compose.

## 🚀 Interacting with the chain {#interacting-with-the-chain}

Since we are using docker images, we can interact with the chain by entering the docker container.

You can see the docker containers running with the gm-world chain and the local DA node by running the following command:

```bash
docker ps
```

You should see output like the following:

```bash
CONTAINER ID   IMAGE      COMMAND                  CREATED          STATUS         PORTS                    NAMES
d50c7f2fffde   local-da   "local-da -listen-all"   10 seconds ago   Up 9 seconds   0.0.0.0:7980->7980/tcp   local-da
b9d5e80e81fb   gm-world   "gmd start --evolve…"   27 minutes ago   Up 9 seconds                            gm-world
```

We can see the gm-world chain running in container `gm-world` and the local DA network running in container `local-da`.

Since our chain is running in a docker container, we want to enter the docker container to interact with it via the command `gmd`. We can do this by running:

```bash
docker exec -it gm-world sh
```

Now that you are in the docker container, you can interact with the chain using the `gmd` command and the example you used in the [gm-world tutorial](/guides/gm-world.md).

Once you are done interacting with your chain, you can exit out of your docker container with:

```bash
exit
```

Then you can shut down your chain environment by running `CRTL+C` in your terminal.

If you want to stop the docker containers without shutting down your terminal, you can run:

```bash
docker compose down
```

## 🎉 Next steps

Congratulations again! You now know how to run your chain with docker compose and interact with it using the `gmd` command in the docker container.
