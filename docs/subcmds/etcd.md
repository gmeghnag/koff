# `koff etcd inspect <etcd_snapshot_file> [<key>]`

`koff etcd inspect` allows to list all the key in a etcd snapshot:
```
$ koff etcd inspect etcd_snap.db | tail -5
/kubernetes.io/configmaps/kube-system/kube-controller-manager
/kubernetes.io/leases/kube-system/kube-controller-manager
/kubernetes.io/configmaps/openshift-kube-scheduler/kube-scheduler
/kubernetes.io/leases/openshift-kube-scheduler/kube-scheduler
/kubernetes.io/apiserver.openshift.io/apirequestcounts/validatingwebhookconfigurations.v1.admissionregistration.k8s.io
```

and to retrieve data from it:
```
$ koff etcd inspect etcd_snap.db /kubernetes.io/configmaps/kube-system/kube-controller-manager -o yaml
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    control-plane.alpha.kubernetes.io/leader: '{"holderIdentity":"ip-192-168-5-44_c49f5ead-c179-44b7-95a0-bbf24ae733fb","leaseDurationSeconds":15,"acquireTime":"2024-02-05T22:46:16Z","renewTime":"2024-02-06T07:29:18Z","leaderTransitions":5}'
  creationTimestamp: "2024-02-05T22:28:17Z"
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:control-plane.alpha.kubernetes.io/leader: {}
    manager: kube-controller-manager
    operation: Update
    time: "2024-02-06T07:29:18Z"
  name: kube-controller-manager
  namespace: kube-system
  uid: 4186c999-6e76-454b-8fa6-9dd4138d70d6
```