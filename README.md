# `koff`
[![GitHub Actions Build Status](https://github.com/gmeghnag/koff/actions/workflows/build.yml/badge.svg)](https://github.com/gmeghnag/koff/actions?query=workflow%3ABuild)
![Downloads](https://img.shields.io/github/downloads/gmeghnag/koff/total)
<img src="./docs/images/preview.png" width="100%">

`koff` is a command-line tool that processes Kubernetes `yaml|json` manifests **or etcd database snapshot** and returns their appropriate tabular format, as well as `yaml` or `json`.

## **¿What?**
It allows you to process Kubernetes resources in `yaml` or `json` format, from either file or piped input or etcd snapshot file.<br />
It reads input, performs the specific filter operations based on the flags and arguments (if provided), and writes the output in either tabular (as default), `json` or `yaml` format. 

## **¿Why?**
Helpful in conjunction with `kubectl` to take a "snapshot" of specific resources at that specific point in time and parse the same later on.

## **¿How?**
- Via piped input:
  ```
  $ cat resources.yaml | koff get pod/postgresql-1-2gxpm svc/postgresql
  NAME                     READY   STATUS    RESTARTS   AGE
  pod/postgresql-1-2gxpm   1/1     Running   0          15m

  NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
  service/postgresql   ClusterIP   172.30.58.223   <none>        5432/TCP   16m
  ```
- Referencing the file to use via `koff use <resources>.yaml` before executing `koff`:
  ```
  $ koff use resources.yaml
  $ koff get pod/postgresql-1-2gxpm svc/postgresql
  NAME                     READY   STATUS    RESTARTS   AGE
  pod/postgresql-1-2gxpm   1/1     Running   0          15m

  NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
  service/postgresql   ClusterIP   172.30.58.223   <none>        5432/TCP   16m
  ```
- Using an etcd snapshot to get the Kubernetes object contained in it (the only requirement is that the database snapshot needs to have `.db` suffix):
  ```
  $ koff use etcd_snap.db

  $ koff get po,svc -n openshift-etcd
  NAME                                                READY   STATUS      RESTARTS   AGE
  pod/etcd-guard-labocp417-jh6n2-master-0          1/1     Running     0          1d
  pod/etcd-guard-labocp417-jh6n2-master-1          1/1     Running     0          1d
  pod/etcd-guard-labocp417-jh6n2-master-2          1/1     Running     0          1d
  pod/etcd-labocp417-jh6n2-master-0                4/4     Running     0          1d
  pod/etcd-labocp417-jh6n2-master-1                4/4     Running     0          1d
  pod/etcd-labocp417-jh6n2-master-2                4/4     Running     0          1d
  pod/installer-8-labocp417-jh6n2-master-2         0/1     Completed   0          1d
  pod/revision-pruner-7-labocp417-jh6n2-master-0   0/1     Completed   0          1d
  pod/revision-pruner-7-labocp417-jh6n2-master-1   0/1     Completed   0          1d
  pod/revision-pruner-7-labocp417-jh6n2-master-2   0/1     Completed   0          1d
  pod/revision-pruner-8-labocp417-jh6n2-master-0   0/1     Completed   0          1d
  pod/revision-pruner-8-labocp417-jh6n2-master-1   0/1     Completed   0          1d
  pod/revision-pruner-8-labocp417-jh6n2-master-2   0/1     Completed   0          1d

  NAME           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)             AGE
  service/etcd   ClusterIP   172.30.24.145   <none>        2379/TCP,9979/TCP   1d
  ```

## Installation
### Using `go`
```
go install github.com/gmeghnag/koff 
```
### Download the latest binary
```
OS=Linux        # or Darwin
curl -sL "https://github.com/gmeghnag/koff/releases/latest/download/koff_${OS}_x86_64.tar.gz" | tar xzf - koff
chmod +x ./koff
```
### Build from source
```
$ git clone https://github.com/gmeghnag/koff.git
$ cd koff/
$ go install
```

## Upgrade
It's possible to upgrade the tool to the latest version by running `koff upgrade --to=latest`


