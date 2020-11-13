#!/bin/bash

NAMESPACE=${1:-default}
echo "Using namespace $NAMESPACE"

kubectl apply -f samples/kubernetes/config-map-plan.yaml --namespace "$NAMESPACE"
kubectl apply -f samples/kubernetes/atlas-service-broker-auth.yaml --namespace "$NAMESPACE"
kubectl apply -f samples/kubernetes/deployment.yaml --namespace "$NAMESPACE"
kubectl apply -f samples/kubernetes/service-broker.yaml --namespace "$NAMESPACE"
kubectl apply -f samples/kubernetes/instance.yaml --namespace "$NAMESPACE"
kubectl apply -f samples/kubernetes/binding.yaml --namespace "$NAMESPACE"
