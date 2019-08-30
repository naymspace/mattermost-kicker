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

Open the [System Console → Plugin Management](http://localhost:8065/admin_console/plugins/plugin_management).

Go to „Upload Plugin“ and upload the built Mattermost plugin archive, click „Upload“.

Scroll down to „Kicker Plugin by naymspace“ and click on „Enable“.

# Usage

In any channel, issue a command like this:

```
/kicker 12 00
```

# Plugin Starter Template ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-starter-template/master.svg)

This plugin serves as a starting point for writing a Mattermost plugin. Feel free to base your own plugin off this repository.

To learn more about plugins, see [our plugin documentation](https://developers.mattermost.com/extend/plugins/).

## Getting Started
Use GitHub's template feature to make a copy of this repository by clicking the "Use this template" button then clone outside of `$GOPATH`.

Alternatively shallow clone the repository to a directory outside of `$GOPATH` matching your plugin name:
```
git clone --depth 1 https://github.com/mattermost/mattermost-plugin-starter-template com.example.my-plugin
```

Note that this project uses [Go modules](https://github.com/golang/go/wiki/Modules). Be sure to locate the project outside of `$GOPATH`, or allow the use of Go modules within your `$GOPATH` with an `export GO111MODULE=on`.

Edit `plugin.json` with your `id`, `name`, and `description`:
```
{
    "id": "com.example.my-plugin",
    "name": "My Plugin",
    "description": "A plugin to enhance Mattermost."
}
```

Build your plugin:
```
make
```

This will produce a single plugin file (with support for multiple architectures) for upload to your Mattermost server:

```
dist/com.example.my-plugin.tar.gz
```

There is a build target to automate deploying and enabling the plugin to your server, but it requires configuration and [http](https://httpie.org/) to be installed:
```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
make deploy
```

Alternatively, if you are running your `mattermost-server` out of a sibling directory by the same name, use the `deploy` target alone to  unpack the files into the right directory. You will need to restart your server and manually enable your plugin.

In production, deploy and upload your plugin via the [System Console](https://about.mattermost.com/default-plugin-uploads).

## Q&A

### How do I make a server-only or web app-only plugin?

Simply delete the `server` or `webapp` folders and remove the corresponding sections from `plugin.json`. The build scripts will skip the missing portions automatically.

### How do I include assets in the plugin bundle?

Place them into the `assets` directory. To use an asset at runtime, build the path to your asset and open as a regular file:

```go
bundlePath, err := p.API.GetBundlePath()
if err != nil {
    return errors.Wrap(err, "failed to get bundle path")
}

profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile_image.png"))
if err != nil {
    return errors.Wrap(err, "failed to read profile image")
}

if appErr := p.API.SetProfileImage(userID, profileImage); appErr != nil {
    return errors.Wrap(err, "failed to set profile image")
}
```
