# `koff`
<img src="./docs/images/preview.png" width="100%">

`koff` is a command-line tool that process kuberntes `yaml` manifests and return their appropriate tabular format, as well as `yaml` or `json`.

## **¿What?**
It allows you to process kubernetes resources in `yaml` or `json` format, from either file or piped input.<br />
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

## Installation
### Using `go`
```
go install github.com/gmeghnag/koff 
```
### Download the latest binary
```
OS=Linux        # or Darwin
curl -sL https://github.com/gmeghnag/koff/releases/latest/download/koff_${OS}_x86_64.tar.gz| tar xzf - koff
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


