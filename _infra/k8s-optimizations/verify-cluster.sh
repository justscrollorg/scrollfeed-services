#!/bin/bash
# Kubernetes Cluster Health and Optimization Verification Script

echo "=========================================="
echo "Kubernetes Cluster Health Check"
echo "=========================================="
echo ""

# Check cluster connectivity
echo "1. Cluster Connectivity:"
kubectl cluster-info | head -2
echo ""

# Check node status
echo "2. Node Status:"
kubectl get nodes
echo ""

# Check node resource usage
echo "3. Node Resource Usage:"
kubectl top nodes
echo ""

# Check HPA status
echo "4. Horizontal Pod Autoscaler Status:"
kubectl get hpa -n default
echo ""

# Check KEDA installation
echo "5. KEDA Status:"
kubectl get pods -n keda
echo ""

# Check metrics-server
echo "6. Metrics Server Status:"
kubectl get deployment metrics-server -n kube-system
echo ""

# Check failed pods
echo "7. Failed/Pending Pods:"
kubectl get pods --all-namespaces --field-selector=status.phase!=Running,status.phase!=Succeeded
echo ""

# Check PDBs
echo "8. Pod Disruption Budgets:"
kubectl get pdb --all-namespaces
echo ""

# Check Priority Classes
echo "9. Priority Classes:"
kubectl get priorityclasses | grep -E "NAME|priority-services|priority-batch"
echo ""

# Pod distribution across nodes
echo "10. Pod Distribution Across Nodes:"
echo "Node                            Pod Count"
echo "----                            ---------"
kubectl get pods --all-namespaces -o wide --field-selector=status.phase=Running | \
  awk 'NR>1 {print $8}' | grep -E '^lke' | sort | uniq -c | awk '{print $2 "  " $1}'
echo ""

# Top memory consuming pods
echo "11. Top 10 Memory Consuming Pods:"
kubectl top pods --all-namespaces --sort-by=memory | head -11
echo ""

# Check for resource-constrained nodes
echo "12. Node Capacity vs Allocation:"
kubectl describe nodes | grep -A 5 "Allocated resources" | grep -E "Resource|cpu|memory|--"
echo ""

# Check recent events
echo "13. Recent Cluster Events (HPA related):"
kubectl get events -n default --field-selector involvedObject.kind=HorizontalPodAutoscaler --sort-by='.lastTimestamp' | tail -10
echo ""

echo "=========================================="
echo "Verification Complete!"
echo "=========================================="
echo ""
echo "Next Steps:"
echo "1. Monitor HPA behavior: kubectl get hpa -n default -w"
echo "2. Apply KEDA ScaledObjects: kubectl apply -f k8s-optimizations/keda-scaledobjects.yaml"
echo "3. Deploy Cluster Autoscaler (after creating Linode API secret)"
echo "4. Apply resource limits during maintenance window"
echo ""
