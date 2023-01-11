# monitoring-cell

## Running the operator

With a running kubernetes cluster, run:

```console
make install
kubectl apply -f config/samples/monitoring_v1alpha1_cell.yaml
make run
```