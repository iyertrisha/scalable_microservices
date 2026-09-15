# AWS EKS deployment runbook

FleetFlow uses the same Kubernetes manifests for kind and EKS. Do not invent a cloud deploy — only run these when AWS credentials and an EKS cluster exist.

## Prerequisites

- `aws` CLI configured (`aws sts get-caller-identity` works)
- `eksctl` or Terraform to create a cluster
- `kubectl` + `helm`

## 1. Create cluster (example)

```bash
eksctl create cluster --name fleetflow --region ap-south-1 --nodegroup-name ng --nodes 3 --node-type t3.medium
aws eks update-kubeconfig --name fleetflow --region ap-south-1
```

## 2. Push images to ECR

```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
REGION=ap-south-1
for SVC in order-service driver-service dispatch-service tracking-service location-service pricing-service notification-service; do
  aws ecr create-repository --repository-name fleetflow/$SVC --region $REGION || true
  docker build --build-arg SERVICE=$SVC -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/fleetflow/$SVC:latest -f deployments/docker/Dockerfile .
  aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com
  docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/fleetflow/$SVC:latest
done
```

## 3. Install add-ons

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx --create-namespace

helm repo add kedacore https://kedacore.github.io/charts
helm install keda kedacore/keda -n keda --create-namespace
```

## 4. Deploy

Patch imagePullPolicy / image names for ECR (overlay), then:

```bash
kubectl apply -k deployments/k8s/overlays/local
kubectl apply -f deployments/k8s/keda/dispatch-scaledobject.yaml
kubectl apply -f deployments/observability/stack.yaml
```

## 5. Verify

```bash
kubectl -n fleetflow get pods,hpa,ingress
curl http://<ingress-lb>/health
```

If AWS credentials are unavailable, keep the kind demo as the verified environment and treat this file as the production path.
