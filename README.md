# Prerequisites

You will need `npm` and `curl`.

# Go

You will need to [install Go](https://golang.org/doc/install). When using a Debian-based system, there may be a package `golang`, which cannot be used, as the Go version is too old.

Build the project by running this command while in the project's root folder:

```shell
make
```

This will create an archive containing the Mattermost plugin, named i.e. `dist/com.naymspace.mattermost-kicker-0.1.0.tar.gz`.

# Mattermost

You will need a locally running Mattermost. See the [installation instructions](https://mattermost.com/download/).

For example using Docker:

```shell
docker run --rm --name mattermost-preview -d --publish 8065:8065 mattermost/mattermost-preview
```

Wait a few seconds, then visit the Mattermost start page: http://localhost:8065/

Register an account (email does not matter, no mails are sent out), then create a team.

Open the [System Console → Web Server](http://localhost:8065/admin_console/environment/web_server).

Enter `http://localhost:8065` into the „Site URL“ field, and hit the „Save“ buttons. This is **required** to make the plugin work.

Open the [System Console → Plugin Management](http://localhost:8065/admin_console/plugins/plugin_management).

Go to „Upload Plugin“ and upload the built Mattermost plugin archive, click „Upload“.

Scroll down to „Kicker Plugin by naymspace“ and click on „Enable“.

# Usage

In any channel, issue a command like this:

```
/kicker 12 00
```

# Deployment

For development, there is a build target to automate deploying and enabling the plugin to your server, but it requires configuration and [http](https://httpie.org/) to be installed:
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

Alternatively, if you are running your `mattermost-server` out of a sibling directory by the same name, use the `deploy` target alone to  unpack the files into the right directory. You will need to restart your server and manually enable your plugin.

In production, deploy and upload your plugin via the [System Console](https://about.mattermost.com/default-plugin-uploads).
