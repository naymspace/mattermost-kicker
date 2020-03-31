# mattermost-kicker

A plugin for [Mattermost](https://mattermost.com/) for kicker matches, creating a poll for quick and fair player picking.

## Releases

For a quick-start you can directly download the lastest release here:
[Release 1.1.2](releases/com.naymspace.mattermost-kicker-1.1.2.tar.gz)

You can also build it yourself:

## Getting Started

### Prerequisites

The Mattermost server must be version 5.12 or newer. For development, you can use a Mattermost server running in a Docker container (see below).

On your development machine will need a current `npm` and any `curl` version.

You will also need to [install Go](https://golang.org/doc/install). When using a Debian-based system, there may be a package `golang`, which cannot be used, as the Go version is too old.

### Installing

[Install Go](https://golang.org/doc/install) if not done already.

You will need a locally running Mattermost. See the [installation instructions](https://mattermost.com/download/).

For example using Docker:

```shell
docker run --name mattermost-preview -d --publish 8065:8065 mattermost/mattermost-preview
```

### Configuration

Wait a few seconds after starting Mattermost, then visit the start page: http://localhost:8065/

Register an account (email does not matter, no mails are sent out), then create a team.

Open the [System Console → Web Server](http://localhost:8065/admin_console/environment/web_server).

Enter `http://localhost:8065` into the „Site URL“ field, and hit the „Save“ buttons. This is **required** to make the plugin work.

### Environment variables

-   **MM_SERVICESETTINGS_SITEURL** [String] – Mattermost server URL, used for deployment; e.g. `http://localhost:8065`
-   **MM_ADMIN_USERNAME** [String] – username for deployment (must have admin privileges)
-   **MM_ADMIN_PASSWORD** [String] – password for deployment

## Running and Deployment

The plugin is automatically started and stopped with Mattermost.

It has to be deployed first to Mattermost, see below.

The Mattermost server Docker container can be started and stopped normally:

```
docker start mattermost-preview
docker stop mattermost-preview
```

### Access application

```
http://localhost:8065
```

In any channel, type `/kicker` to see the plugin's `kicker` command and the available options.

Example: to start a kicker match at 12:00, use

```
/kicker 12 00
```

### Building and Deployment

Build the project by running this command while in the project's root folder:

```shell
make
```

This will create an archive containing the Mattermost plugin, named i.e. `dist/com.naymspace.mattermost-kicker-0.1.0.tar.gz`.

You can deploy the plugin manually or (for development) with the `Makefile`.

#### Manual deployment

Open the [System Console → Plugin Management](http://localhost:8065/admin_console/plugins/plugin_management).

Go to „Upload Plugin“ and upload the built Mattermost plugin archive, click „Upload“.

Scroll down to „Kicker Plugin by naymspace“ and click on „Enable“.

#### Automated deployment

This can only be done in the development environment.

After setting up the required environment variables (see above), use

```
make deploy
```

## Automatic tests

Tests are automatically run when executing `make`.

You can also run the tests without compiling the plugin:

```
make test
```

## Important dependencies and documentation

-   [mattermost plugin API documentation](https://developers.mattermost.com/extend/plugins/server/reference/#API)
-   [mattermost plugin model documentation](https://godoc.org/github.com/mattermost/mattermost-server/model)
-   [Mattermost API](https://api.mattermost.com/)
-   [Mattermost interactive messages examples](https://docs.mattermost.com/developer/interactive-messages.html)
