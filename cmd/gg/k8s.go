package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/internal/clioutput"
	"github.com/spf13/cobra"
)

// k8sCmd represents the k8s command
var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Kubernetes deployment management",
	Long: `Kubernetes deployment management for the application.

This command provides subcommands to:
- Generate Kubernetes deployment manifests
- Apply configurations to Kubernetes cluster
- Delete resources from Kubernetes cluster

Examples:
  gg k8s gen                   # Generate K8s manifests
  gg k8s apply                 # Apply manifests to cluster
  gg k8s delete                # Delete resources from cluster`,
}

// k8sGenCmd represents the k8s gen command
var k8sGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate Kubernetes deployment manifests",
	Long: `Generate Kubernetes deployment manifests including:
- Deployment with health checks and resource limits
- Service (ClusterIP and NodePort)
- ConfigMap for application configuration
- Optional Ingress configuration

Examples:
  gg k8s gen                   # Generate all manifests
  gg k8s gen --namespace prod  # Generate with custom namespace
  gg k8s gen --replicas 3      # Generate with 3 replicas`,
	RunE: k8sGenRun,
}

// k8sApplyCmd represents the k8s apply command
var k8sApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply Kubernetes manifests to cluster",
	Long: `Apply the generated Kubernetes manifests to the cluster.

This command will apply all manifest files in the k8s directory.

Examples:
  gg k8s apply                 # Apply all manifests
  gg k8s apply --namespace prod # Apply to specific namespace`,
	RunE: k8sApplyRun,
}

// k8sDeleteCmd represents the k8s delete command
var k8sDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete Kubernetes resources from cluster",
	Long: `Delete the Kubernetes resources from the cluster.

This command will delete all resources defined in the k8s directory.

Examples:
  gg k8s delete                # Delete all resources
  gg k8s delete --namespace prod # Delete from specific namespace`,
	RunE: k8sDeleteRun,
}

// k8sDeployCmd represents the k8s deploy command
var k8sDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Generate and deploy Kubernetes manifests to cluster",
	Long: `Generate Kubernetes manifests and deploy them to the cluster in one command.

This command combines the functionality of 'gen' and 'apply' commands:
1. Generate all Kubernetes manifests (Deployment, Service, ConfigMap, optional Ingress)
2. Apply the generated manifests to the specified cluster

Examples:
  gg k8s deploy                    # Generate and deploy with defaults
  gg k8s deploy --namespace prod   # Deploy to production namespace
  gg k8s deploy --replicas 3       # Deploy with 3 replicas
  gg k8s deploy --ingress          # Deploy with Ingress configuration`,
	RunE: k8sDeployRun,
}

var (
	k8sNamespace string
	k8sReplicas  int32
	k8sPort      int32
	k8sIngress   bool
)

func init() {
	// Add subcommands to k8s command
	k8sCmd.AddCommand(k8sGenCmd, k8sApplyCmd, k8sDeleteCmd, k8sDeployCmd)

	// Flags for k8s gen command
	k8sGenCmd.Flags().StringVarP(&k8sNamespace, "namespace", "n", "default", "Kubernetes namespace")
	k8sGenCmd.Flags().Int32Var(&k8sReplicas, "replicas", 1, "Number of replicas")
	k8sGenCmd.Flags().Int32VarP(&k8sPort, "port", "p", 8080, "Application port")
	k8sGenCmd.Flags().BoolVar(&k8sIngress, "ingress", false, "Generate Ingress configuration")

	// Flags for k8s apply command
	k8sApplyCmd.Flags().StringVarP(&k8sNamespace, "namespace", "n", "default", "Kubernetes namespace")

	// Flags for k8s delete command
	k8sDeleteCmd.Flags().StringVarP(&k8sNamespace, "namespace", "n", "default", "Kubernetes namespace")

	// Flags for k8s deploy command
	k8sDeployCmd.Flags().StringVarP(&k8sNamespace, "namespace", "n", "default", "Kubernetes namespace")
	k8sDeployCmd.Flags().Int32Var(&k8sReplicas, "replicas", 1, "Number of replicas")
	k8sDeployCmd.Flags().Int32VarP(&k8sPort, "port", "p", 8080, "Application port")
	k8sDeployCmd.Flags().BoolVar(&k8sIngress, "ingress", false, "Generate Ingress configuration")
}

// k8sGenRun generates Kubernetes manifests
func k8sGenRun(cmd *cobra.Command, args []string) error {
	clioutput.Section("Generate Kubernetes Manifests")

	// Create k8s directory if it doesn't exist
	k8sDir := "k8s"
	if err := ensureParentDir(filepath.Join(k8sDir, "dummy")); err != nil {
		return fmt.Errorf("failed to create k8s directory: %w", err)
	}

	// Get module name for app name
	moduleName, err := getModuleName()
	if err != nil {
		return fmt.Errorf("failed to get module name: %w", err)
	}

	appName := filepath.Base(moduleName)
	appName = strings.ToLower(strings.ReplaceAll(appName, "_", "-"))

	// Generate ConfigMap
	if err := generateConfigMap(k8sDir, appName); err != nil {
		return fmt.Errorf("failed to generate ConfigMap: %w", err)
	}

	// Generate Deployment
	if err := generateDeployment(k8sDir, appName); err != nil {
		return fmt.Errorf("failed to generate Deployment: %w", err)
	}

	// Generate Service
	if err := generateService(k8sDir, appName); err != nil {
		return fmt.Errorf("failed to generate Service: %w", err)
	}

	// Generate Ingress if requested
	if k8sIngress {
		if err := generateIngress(k8sDir, appName); err != nil {
			return fmt.Errorf("failed to generate Ingress: %w", err)
		}
	}

	clioutput.Success("", "Kubernetes manifests generated successfully in %s/", k8sDir)
	return nil
}

// k8sApplyRun applies Kubernetes manifests to cluster
func k8sApplyRun(cmd *cobra.Command, args []string) error {
	// Check if kubectl is installed
	if err := checkKubectlInstalled(); err != nil {
		return fmt.Errorf("kubectl is not installed or not accessible: %w", err)
	}

	clioutput.Section("Apply Kubernetes Manifests")

	k8sDir := "k8s"
	if !fileExists(k8sDir) {
		return errors.New("k8s directory not found. Run 'gg k8s gen' first")
	}

	// Apply all YAML files in k8s directory
	execCmd := exec.Command("kubectl", "apply", "-f", k8sDir, "--namespace", k8sNamespace)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		clioutput.Error("", "Failed to apply Kubernetes manifests: %v", err)
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	clioutput.Success("", "Kubernetes manifests applied successfully to namespace: %s", k8sNamespace)
	return nil
}

// k8sDeleteRun deletes Kubernetes resources from cluster
func k8sDeleteRun(cmd *cobra.Command, args []string) error {
	// Check if kubectl is installed
	if err := checkKubectlInstalled(); err != nil {
		return fmt.Errorf("kubectl is not installed or not accessible: %w", err)
	}

	clioutput.Section("Delete Kubernetes Resources")

	k8sDir := "k8s"
	if !fileExists(k8sDir) {
		return errors.New("k8s directory not found. Nothing to delete")
	}

	// Delete all resources defined in k8s directory
	execCmd := exec.Command("kubectl", "delete", "-f", k8sDir, "--namespace", k8sNamespace, "--ignore-not-found=true")
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		clioutput.Error("", "Failed to delete Kubernetes resources: %v", err)
		return fmt.Errorf("kubectl delete failed: %w", err)
	}

	clioutput.Success("", "Kubernetes resources deleted successfully from namespace: %s", k8sNamespace)
	return nil
}

// checkKubectlInstalled checks if kubectl is installed
func checkKubectlInstalled() error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		clioutput.Warn("", "kubectl is not installed or not in PATH")
		return errors.New("kubectl command not found")
	}
	return nil
}

// generateConfigMap generates a ConfigMap manifest
func generateConfigMap(k8sDir, appName string) error {
	configMapPath := filepath.Join(k8sDir, "configmap.yaml")

	configMapContent := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
  labels:
    app: %s
data:
  # Application configuration
  APP_ENV: "production"
  LOG_LEVEL: "info"
  PORT: "%d"
  # Database configuration (example)
  # DB_HOST: "postgres-service"
  # DB_PORT: "5432"
  # DB_NAME: "myapp"
`, appName, k8sNamespace, appName, k8sPort)

	writeFileWithLog(configMapPath, configMapContent)

	return nil
}

// generateDeployment generates a Deployment manifest
func generateDeployment(k8sDir, appName string) error {
	deploymentPath := filepath.Join(k8sDir, "deployment.yaml")

	deploymentContent := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        ports:
        - containerPort: %d
          name: http
        env:
        - name: PORT
          valueFrom:
            configMapKeyRef:
              name: %s-config
              key: PORT
        - name: APP_ENV
          valueFrom:
            configMapKeyRef:
              name: %s-config
              key: APP_ENV
        - name: LOG_LEVEL
          valueFrom:
            configMapKeyRef:
              name: %s-config
              key: LOG_LEVEL
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          runAsUser: 1001
          capabilities:
            drop:
            - ALL
      securityContext:
        fsGroup: 1001
`, appName, k8sNamespace, appName, k8sReplicas, appName, appName, appName, appName, k8sPort, appName, appName, appName)

	writeFileWithLog(deploymentPath, deploymentContent)

	return nil
}

// generateService generates a Service manifest
func generateService(k8sDir, appName string) error {
	servicePath := filepath.Join(k8sDir, "service.yaml")

	serviceContent := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s-service
  namespace: %s
  labels:
    app: %s
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app: %s
---
apiVersion: v1
kind: Service
metadata:
  name: %s-nodeport
  namespace: %s
  labels:
    app: %s
spec:
  type: NodePort
  ports:
  - port: 80
    targetPort: http
    protocol: TCP
    name: http
    nodePort: 30080
  selector:
    app: %s
`, appName, k8sNamespace, appName, appName, appName, k8sNamespace, appName, appName)

	writeFileWithLog(servicePath, serviceContent)

	return nil
}

// generateIngress generates an Ingress manifest
func generateIngress(k8sDir, appName string) error {
	ingressPath := filepath.Join(k8sDir, "ingress.yaml")

	ingressContent := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s-ingress
  namespace: %s
  labels:
    app: %s
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
spec:
  ingressClassName: nginx
  rules:
  - host: %s.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s-service
            port:
              number: 80
  # Uncomment for HTTPS
  # tls:
  # - hosts:
  #   - %s.local
  #   secretName: %s-tls
`, appName, k8sNamespace, appName, appName, appName, appName, appName)

	writeFileWithLog(ingressPath, ingressContent)

	return nil
}

// k8sDeployRun generates and deploys Kubernetes manifests to cluster
func k8sDeployRun(cmd *cobra.Command, args []string) error {
	// Check if kubectl is installed first
	if err := checkKubectlInstalled(); err != nil {
		return fmt.Errorf("kubectl is not installed or not accessible: %w", err)
	}

	clioutput.Section("Deploy to Kubernetes Cluster")

	// Step 1: Generate manifests
	clioutput.Info("", "Step 1/2: Generating Kubernetes manifests...")
	if err := k8sGenRun(cmd, args); err != nil {
		clioutput.Error("", "Failed to generate Kubernetes manifests: %v", err)
		return fmt.Errorf("manifest generation failed: %w", err)
	}
	clioutput.Success("", "Manifests generated successfully")

	// Step 2: Apply manifests to cluster
	clioutput.Info("", "Step 2/2: Applying manifests to cluster (namespace: %s)...", k8sNamespace)

	k8sDir := "k8s"
	if !fileExists(k8sDir) {
		return errors.New("k8s directory not found after generation")
	}

	// Apply all YAML files in k8s directory
	execCmd := exec.Command("kubectl", "apply", "-f", k8sDir, "--namespace", k8sNamespace)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		clioutput.Error("", "Failed to apply Kubernetes manifests: %v", err)
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	clioutput.Success("", "Deployment completed successfully")
	clioutput.Info("", "Application deployed to namespace: %s", k8sNamespace)
	clioutput.Info("", "Replicas: %d, Port: %d", k8sReplicas, k8sPort)
	if k8sIngress {
		clioutput.Info("", "Ingress configuration included")
	}

	return nil
}
