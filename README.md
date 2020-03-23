# git-remote-https+iap

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/adohkan/git-remote-https-iap)](https://github.com/adohkan/git-remote-https-iap/releases/latest)
[![GitHub](https://img.shields.io/github/license/adohkan/git-remote-https-iap)](LICENSE.txt)
[![Go Report Card](https://goreportcard.com/badge/github.com/adohkan/git-remote-https-iap)](https://goreportcard.com/report/github.com/adohkan/git-remote-https-iap)

An open source [`git-remote-helper`](https://git-scm.com/docs/git-remote-helpers) that handles authentication for [GCP Identity Aware Proxy](https://cloud.google.com/iap).

## Getting Started

### Installing

- Download pre-compiled binaries from [`our release page`](https://github.com/adohkan/git-remote-https-iap/releases/latest).
- Install `git-remote-https+iap` binary onto the system `$PATH`
- Run `GIT_IAP_VERBOSE=1 git-remote-https+iap install`

### Configuring

- [Generate OAuth credentials FOR THE HELPER](https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_desktop_app)[1]
- Configure the IAP protected repositories:

```
git-remote-https+iap configure \
  --repoURL=https://git.domain.acme/demo/hello-world.git \
  --helperID=xxx \
  --helperSecret=yyy \
  --clientID=zzz
```

**Notes**:
* In the example above, `xxx` and `yyy` are the OAuth credentials FOR THE HELPER, that needs to be created as instructed [here](https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_desktop_app). `zzz` is the OAuth client ID that has been created when your Identity Aware Proxy instance has been created.
* All repositories served on the same domain (`git.domain.acme`) would share the same configuration


[1]: This needs to be done only once per _organisation_. While [these credentials are not treated as secret](https://developers.google.com/identity/protocols/oauth2#installed) and can be shared within your organisation, [it seem forbidden to publish them in any open source project](https://stackoverflow.com/questions/27585412/can-i-really-not-ship-open-source-with-client-id).

### Usage

Once your domain has been configured, you should be able to use `git` as you would normally do, without thinking about the IAP layer.

```
$ git clone https://git.domain.acme/demo/hello-world.git
```

> If you are using [`git-lfs`](https://git-lfs.github.com/), the minimal version requirement is [`>= v2.9.0`](https://github.com/git-lfs/git-lfs/releases/), which introduced support of HTTP cookies.

### Troubleshoot

If needed, you can set the `GIT_IAP_VERBOSE=1` environment variable in order to increase the verbosity of the logs.
