package main
import (
  "dagger.io/dagger"
  "dagger.io/dagger/core"
  "universe.dagger.io/docker"
  "universe.dagger.io/docker/cli"
)

#GoTrueBuild: {
  pull: core.#GitPull & {
    remote: "https://github.com/FictionOS/gotrue.git"
    ref: "master"
    keepGitDir: false
  }
  build: docker.#Build & {
    steps: [
      docker.#Dockerfile & {
        source: pull.output
      }
    ]
  }
}

#ReadVersion: {
  dir: dagger.#FS

  read: core.#ReadFile & {
    input: dir
    path: "fiction_gotrue_version.txt"
  }

  version: read.contents
}

dagger.#Plan & {
  actions: {
    _goTrue: #GoTrueBuild
    _readVersion: #ReadVersion & {
      dir: client.filesystem.".".read.contents
    }

    // Run GoTrue image locally
    dev: cli.#Load & {
      image: _goTrue.build.output
      host: client.network."unix:///var/run/docker.sock".connect
      tag: "fiction/gotrue"
    }
    // Push GoTrue to Docker Hub
    deploy: {
      _dockerUsername: "fictionhq"
      versionRelease: docker.#Push & {
        "image": _goTrue.build.output
        dest: "\(_dockerUsername)/gotrue:\(_readVersion.version)"
        auth: {
          username: _dockerUsername
          secret: client.env.DOCKER_ACCESS_TOKEN
        }
      }
      latestRelease: docker.#Push & {
        "image": _goTrue.build.output
        dest: "\(_dockerUsername)/gotrue:latest"
        auth: {
          username: _dockerUsername
          secret: client.env.DOCKER_ACCESS_TOKEN
        }
      }
    }
  }
  client: {
    network: "unix:///var/run/docker.sock": connect: dagger.#Socket
    env: DOCKER_ACCESS_TOKEN: dagger.#Secret
    filesystem: '.' : {
      read: contents: dagger.#FS
    }
  }
}