# `koff`

```
# kubectl get node,pod,svc -n kube-system -o yaml > resources.yaml
# cat resources.yaml | koff -N -K
NAME                          STATUS   ROLES           AGE    VERSION
node/kubernetes.example.com   Ready    control-plane   226d   v1.25.2

NAMESPACE     NAME                                                 READY   STATUS    RESTARTS   AGE
kube-system   pod/coredns-565d847f94-8v782                         1/1     Running   3          226d
kube-system   pod/coredns-565d847f94-95nb2                         1/1     Running   3          226d
kube-system   pod/etcd-kubernetes.example.com                      1/1     Running   1          226d
kube-system   pod/kube-apiserver-kubernetes.example.com            1/1     Running   1          226d
kube-system   pod/kube-controller-manager-kubernetes.example.com   1/1     Running   4          226d
kube-system   pod/kube-proxy-kg6g9                                 1/1     Running   0          226d
kube-system   pod/kube-scheduler-kubernetes.example.com            1/1     Running   4          226d

NAMESPACE     NAME               TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)                        AGE
kube-system   service/kube-dns   ClusterIP   172.16.1.10   <none>        53/UDP,53/TCP,9153/TCP         226d
kube-system   service/kubelet    ClusterIP   None          <none>        10250/TCP,10255/TCP,4194/TCP   224d
```
