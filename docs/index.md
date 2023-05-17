# `koff` :fontawesome-brands-golang:
---

## **¿What?**
`koff` is a command-line tool that allows you to process kubernetes resources in `yaml` or `json` format, from either file or piped input.<br />
It reads input, performs the specific filter operations based on the flags and arguments (if provided), and writes the output in either tabular (as default), `json` or `yaml` format. 

```
$ kubectl get pod,svc,ep -o yaml | koff
NAME                                      READY   STATUS      RESTARTS   AGE
pod/postgresql-1-2gxpm                    1/1     Running     0          15m
pod/postgresql-1-deploy                   0/1     Completed   0          16m
pod/rails-postgresql-example-1-build      0/1     Completed   0          16m
pod/rails-postgresql-example-1-deploy     0/1     Completed   0          11m
pod/rails-postgresql-example-1-hook-pre   0/1     Completed   0          11m
pod/rails-postgresql-example-1-v89ph      1/1     Running     0          11m

NAME                               TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/postgresql                 ClusterIP   172.30.58.223   <none>        5432/TCP   16m
service/rails-postgresql-example   ClusterIP   172.30.114.85   <none>        8080/TCP   16m

NAME                                 ENDPOINTS           AGE
endpoints/postgresql                 10.129.3.75:5432    16m
endpoints/rails-postgresql-example   10.128.2.184:8080   16m
```

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
???+ tip "Dealing with Custom Resources"

    To return a [*custom resource*](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) table in its default tabular format it's needed to add the respective `CustomResourceDefinition` `yaml` manifest into the path `~/.koff/customresourcedefinition`.<br />
    As default behaviour, if the `CustomResourceDefinition` for the *custom resource* is not found the resulting table will have three columns only: `NAMESPACE`, `NAME`, `CREATED AT`.

    ```
    $ kubectl get clusteroperator etcd -o yaml | koff 
    NAME                                       CREATED AT
    clusteroperator.config.openshift.io/etcd   2023-05-15T01:53:20

    $ kubectl get crd clusteroperators.config.openshift.io -o yaml > ~/.koff/customresourcedefinitions/clusteroperators.config.openshift.io.yaml

    $ kubectl get clusteroperator etcd -o yaml | koff
    NAME                                       VERSION   AVAILABLE   PROGRESSING   DEGRADED   SINCE
    clusteroperator.config.openshift.io/etcd   4.9.59    True        False         False      2d
    ```
