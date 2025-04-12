# Helm Whatup

This is a Helm plugin to help users determine if there's an update available for their installed charts. It works by reading your locally cached index files from the chart repositories (via `helm repo update`) and checking the version against the latest deployed version of your charts in the Kubernetes cluster.

## Compatibility

- Helm v3.x (fully compatible)
- ~~Helm v2.x~~ (no longer supported)

## Supported Architectures

- Linux (amd64, arm64)
- macOS (NO Intel Support, arm64)

## Usage

```
 helm repo update
 helm whatup
```

## Install

```
 helm plugin install https://github.com/lucas-albers-lz4/helm-whatup
```

The above will fetch the latest binary release of `helm whatup` and install it.

### Developer (From Source) Install

If you would like to handle the build yourself, instead of fetching a binary,
this is how recommend doing it.

First, set up your environment:

- You need to have [Go](http://golang.org) installed (version 1.24+). Make sure to set `$GOPATH`
- This project uses Go modules for dependency management

Clone this repo into your `$GOPATH` using git.

```
mkdir -p $GOPATH/src/github.com/lucas-albers-lz4
cd $GOPATH/src/github.com/lucas-albers-lz4
git clone https://github.com/lucas-albers-lz4/helm-whatup
```

Then run the following to get running.

```
 cd helm-whatup
 make bootstrap build
 SKIP_BIN_INSTALL=1 helm plugin install $GOPATH/src/github.com/lucas-albers-lz4/helm-whatup
```

That last command will skip fetching the binary install and use the one you built.

## Running Tests

The project includes unit tests that can be run with:

```
 make test
```

For code linting, you can use:

```
 make lint
```
