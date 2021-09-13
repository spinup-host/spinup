# Contributing to Spinup Host

Thanks for your interest in improving the project! This document provides a step-by-step guide for general contributions to Spinup.

## Communications

We have a slack, join using the invite [link](https://join.slack.com/t/spinuphost/shared_invite/zt-vpe8u3rm-I3UTvHqzNBq0JHWjEQ1Y9Q).

## Submitting a PR

If you have a specific idea of a fix or update, follow these steps below to submit a PR:

- [Contributing to Spiup Host](#contributing-to-spinup-host)
  - [Communications](#communications)
  - [Submitting a PR](#submitting-a-pr)
    - [Step 1: Make the change](#step-1-make-the-change)
    - [Step 2: Start Spinup locally](#step-2-start-spinup-locally)
    - [Step 3: Commit and push your changes](#step-3-commit-and-push-your-changes)
    - [Step 4: Create a pull request](#step-4-create-a-pull-request)
    - [Step 5: Get a code review](#step-5-get-a-code-review)

### Step 1: Make the change

1. Fork the Spinup repo, and then clone it:

   ```bash
   export user={your github. profile name}
   git clone git@github.com:${user}/spinup.git
   ```

2. Set your cloned local to track the upstream repository:

   ```bash
   cd spinup
   git remote add upstream https://github.com/spinup-host/spinup
   ```

3. Disable pushing to upstream master:

   ```bash
   git remote set-url --push upstream no_push
   git remote -v
   ```

   The output should look like:

   ```bash
   origin    git@github.com:$(user)/spinup.git (fetch)
   origin    git@github.com:$(user)/spinup.git (push)
   upstream  https://github.com/spinup-host/spinup (fetch)
   upstream  no_push (push)
   ```

4. Get your local master up-to-date and create your working branch:

   ```bash
   git fetch upstream
   git checkout master
   git rebase upstream/master
   git checkout -b myfeature
   ```

### Step 2: Start Spinup locally

[Local-steup](https://github.com/spinup-host/spinup#how-to-run)

### Step 3: Commit and push your changes

1. Run the following commands to keep your branch in sync:

   ```bash
   git fetch upstream
   git rebase upstream/master
   ```

2. Commit your changes:

   ```bash
   git add -A
   git commit --signoff
   ```

3. Push your changes to the remote branch:

   ```bash
   git push -f origin myfeature
   ```

### Step 4: Create a pull request

1. Visit your fork at GitHub.
2. Click the Compare & pull request button next to your `myfeature` branch.
3. Edit the description of the pull request to match your changes.

### Step 5: Get a code review

Once your pull request has been opened, it will be assigned to at least two reviewers. The reviewers will do a thorough code review of correctness, bugs, opportunities for improvement, documentation and comments, and style.

Commit changes made in response to review comments to the same branch on your fork.
