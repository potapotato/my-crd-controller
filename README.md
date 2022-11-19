# my-crd-controller

## For handmade controller
```bash
make run
```

## For kubebuilder controller

- if you use docker 
```bash
make docker-build
docker save -o hb.tar ${IMG}
scp hb.tar root@node1:/tmp
scp hb.tar root@node2:/tmp
ssh node1 "cd /tmp; ctr -n k8s.io image import hb.tar"
ssh node2 "cd /tmp; ctr -n k8s.io image import hb.tar"
kubectl apply -f config/crd/bases/kb.crd.dango.io_websites.yaml
kubectl apply -f config/samples/kb_v1_website.yaml
kubectl apply -f config/samples/controller.yaml
```

- if you use containerd
```bash
make docker-build
docker save -o hb.tar ${IMG}
ctr -n k8s.io image import hb.tar
scp hb.tar root@node1:/tmp
scp hb.tar root@node2:/tmp
ssh node1 "cd /tmp; ctr -n k8s.io image import hb.tar"
ssh node2 "cd /tmp; ctr -n k8s.io image import hb.tar"
kubectl apply -f config/crd/bases/kb.crd.dango.io_websites.yaml
kubectl apply -f config/samples/kb_v1_website.yaml
kubectl apply -f config/samples/controller.yaml
```

then you can assess the http://masterIp:30010/formyrose/index.html
