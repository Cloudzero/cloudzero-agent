# Test driving workflows

To run the GitHub Actions workflows locally using the act utility, you can follow these steps:

## 1. Get the `act` tool

You can install act using Homebrew on macOS, or download it directly for other platforms.

```sh
brew install act
```

Or download it from the [GitHub releases page](https://github.com/nektos/act).

## 2. Run the workflows manually using act:

act allows you to simulate the GitHub Actions environment and execute the workflows as if they were running on GitHub.

**_Note - for the following commands, it is assumed you are in the base directory of the repository - and have the following environment variables set:_**

* `GH_USER` - set to your github user name (such as josephbarnett)
* `GH_PAT` - set to your github personal access token. This token should have repo write permissions, and package write permissions

Now, you can run the following workflow simulations.

### Manually Trigger the Merge Workflow

The manual merge workflow [release-to-main.yml](release-to-main.yml) will perform a sync merge from the `develop` branch to `main`. 

All releases are based on `main`, where as `develop` is incrementally changing until we are ready to release.

To manual trigger the workflow, use the following command:

```sh
act --container-architecture linux/arm64 \
    -a $GH_USER --secret GITHUB_TOKEN=$GH_PAT \
    -j release-to-main
```

### Manually Trigger the DockerBuild Workflow:

For the DockerBuild workflow [docker-build.yml](docker-build.yml), simulate a push to the main branch, develop branch, and a new release tag. It will automatically build a docker image from the repository code, scan it for security vulnerabilties, then if on main or develop - will publish the docker image to the public GHCR repository associated with the repository. 

The following sub-sections allow you to simulate these events and run each permuation.

#### 1. Simulate a Push to develop Branch

> Don't forget to update the `.json` file first before running the command!

```sh
act --container-architecture linux/arm64 \
    -a $GH_USER --secret GITHUB_TOKEN=$GH_PAT \
    -j docker --eventpath .github/workflows/events/develop-push-event.json
```

#### 2. Simulate a Push to main Branch

> Don't forget to update the `.json` file first!

```sh
act --container-architecture linux/arm64 \
    -a $GH_USER --secret GITHUB_TOKEN=$GH_PAT \
    -j docker --eventpath .github/workflows/events/main-push-event.json
```

#### 3. Simulate a Release Event

> Don't forget to update the `.json` file first!

```sh
act --container-architecture linux/arm64 \
    -a $GH_USER --secret GITHUB_TOKEN=$GH_PAT \
    -j docker --eventpath .github/workflows/events/release-event.json
```


By using these commands, you can test your workflows locally and verify their functionality before pushing them to GitHub. This ensures that your workflows are working correctly without needing to trigger them on the actual repository.