go-repoowners
===
[![Build Status](https://travis-ci.org/nasa9084/go-repoowners.svg?branch=master)](https://travis-ci.org/nasa9084/go-repoowners)

`repoowners` treats OWNERS file and OWNERS_ALIASES file inspired by [kubernetes/test-infra/prow/repoowners](https://github.com/kubernetes/test-infra/prow/repoowners).

## OWNERS spec

OWNERS file defines the approvers and the reviewers for the directory. It is applied to everything within the dir, including the OWNERS file itself and children.
OWNERS files do not have its extension, however, they are written in YAML format and the keys are:

* `approvers`: a list of GitHub usernames or aliases that can approve a PR.
* `reviewers`: a list of GitHub usernames or aliases that can review a PR.
* `required_reviewers`: a list of GitHub usernames or aliases that can review a PR and a review by one of them is required.
* options: a map of options.
  * no_inherit: boolean value which shows exclude parent OWNERS files for the directory and children.

A typical OWNERS file looks like:

``` yaml
---
approvers:
  - alice
  - bob
reviewers:
  - charlie
  - dave
  - ellen
```

In this case, `alice` and `bob` can approve/merge a PR and `charlie`, `dave`, and `ellen` can review a PR.
The GitHub usernames are case-insensitive.

## OWNERS_ALIAS spec

Each repository may contain an OWNERS_ALIAS file at its repository root.
OWNERS_ALIAS file defines groups of GitHub users, they are written in YAML format.
This file contains only one key: `aliases`, a mapping of alias name to a lis of GitHub usernames.

A typical OWNERS_ALIAS file looks like:

``` yaml
---
aliases
  admins:
    - alice
    - bob
  members:
    - charlie
    - dave
    - ellen
```

Then OWNERS can be written:

``` yaml
---
approvers:
  - admins
reviewers:
  - members
```

The alias names and GitHub usernames are case-insensitive.
