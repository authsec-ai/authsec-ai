import { useState, useMemo } from "react";
import { Button } from "../../../components/ui/button";
import { Card, CardContent } from "../../../components/ui/card";
import { PageHeader } from "@/components/layout/PageHeader";
import { HeaderCard, TableCard } from "@/theme/components/cards";
import { Badge } from "../../../components/ui/badge";
import {
  RefreshCw,
  Server,
  CheckCircle,
  XCircle,
  Clock,
  ArrowLeft,
  Activity,
  HelpCircle,
  X,
  Code2,
  Check,
  Copy,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { SessionManager } from "../../../utils/sessionManager";
import {
  useListAgentsQuery,
  type AgentRecord,
} from "../../../app/api/workloadsApi";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import {
  SPIRE_FAQ_DATA,
  type FAQItem,
} from "@/features/wizards/components/spire-faq-data";

type AgentStats = {
  active: number;
  inactive: number;
  total: number;
};

const SPRIRE_FAQ_DATA: FAQItem[] = [
  {
    id: "1",
    question: "How do I deploy my agent and workloads on kubernetes cluster?",
    description: "Learn how to deploy SPIRE agents using Helm, manual setup",
    code: {
      python: [
        {
          methodName: "Method 1: Helm Chart (Recommended)",
          steps: [
            {
              label: "Step 1: Add Helm Repository",
              code: `# Add AuthSec Helm repo
helm repo add authsec https://charts.authsec.ai
helm repo update`,
            },
            {
              label: "Step 2: Create values.yaml",
              code: `cat > icp-agent-values.yaml <<EOF
# ICP Agent Configuration
image:
  repository: your-docker-registry.example.com/icp-agent
  tag: latest
  pullPolicy: Always

# Agent settings
agent:
  tenantId: "your-tenant-id-here"
  clusterName: "my-k8s-cluster"
  icpServiceUrl: "https://your-icp-server.example.com/spiresvc"
  logLevel: info
  socketPath: /run/spire/sockets/agent.sock

# Service Account
serviceAccount:
  create: true
  name: icp-agent

# Security Context
securityContext:
  runAsUser: 0
  runAsGroup: 0
  runAsNonRoot: false
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
    add:
      - SYS_PTRACE  # Required for process attestation
  seccompProfile:
    type: RuntimeDefault

# Resources
resources:
  limits:
    cpu: "500m"
    memory: "512Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"

# Health probes
healthProbe:
  enabled: true
  port: 8080
  livenessProbe:
    initialDelaySeconds: 30
    periodSeconds: 60
    timeoutSeconds: 10
    failureThreshold: 3
  readinessProbe:
    initialDelaySeconds: 10
    periodSeconds: 30
    timeoutSeconds: 5
    failureThreshold: 3

# Tolerations (run on all nodes)
tolerations:
  - operator: Exists

# Node selector (optional - restrict to specific nodes)
nodeSelector: {}
  # role: worker

# Affinity (optional)
affinity: {}
EOF`,
            },
            {
              label: "Step 3: Install Agent",
              code: `# Install in default namespace
helm install icp-agent authsec/icp-agent \
  -f icp-agent-values.yaml \
  --namespace default \
  --create-namespace

# Wait for DaemonSet to be ready
kubectl rollout status daemonset/icp-agent -n default`,
            },
            {
              label: "Step 4: Verify Installation",
              code: `# Check DaemonSet
kubectl get daemonset -n default

# Check pods (should be 1 per node)
kubectl get pods -n default -l app=icp-agent -o wide

# Check logs
kubectl logs -n default -l app=icp-agent --tail=50

# Check health
kubectl exec -n default -l app=icp-agent -- curl http://localhost:8080/healthz`,
            },
          ],
        },
        {
          methodName: "Method 2: kubectl (Manual Deployment)",
          steps: [
            {
              label: "Step 1: Create Namespace",
              code: `kubectl create namespace default`,
            },
            {
              label: "Step 2: Deploy ConfigMap",
              code: `kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: icp-agent-config
  namespace: default
  labels:
    app: icp-agent
data:
  config.yaml: |
    agent:
      tenant_id: "your-tenant-id-here"
      node_id: "\${NODE_NAME}"
      data_dir: "/var/lib/icp-agent"
      socket_path: "/run/spire/sockets/agent.sock"
      renewal_threshold: "6h"

    icp_service:
      address: "https://stage.api.authsec.dev/spiresvc"
      trust_bundle_path: "/etc/icp-agent/ca-bundle.pem"
      timeout: 30
      max_retries: 3
      retry_backoff: 5

    attestation:
      type: "kubernetes"
      kubernetes:
        token_path: "/var/run/secrets/kubernetes.io/serviceaccount/token"
        cluster_name: "my-k8s-cluster"
      unix:
        method: "procfs"

    security:
      cache_encryption_key: ""
      cache_path: "/var/lib/icp-agent/cache/svid.cache"

    logging:
      level: "info"
      format: "json"
      file_path: ""

    health:
      enabled: true
      port: 8080
      bind_address: "0.0.0.0"
EOF`,
            },
            {
              label: "Step 3: Deploy RBAC",
              code: `kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: icp-agent
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: icp-agent
rules:
  - apiGroups: [""]
    resources: ["pods", "nodes"]
    verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: icp-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: icp-agent
subjects:
  - kind: ServiceAccount
    name: icp-agent
    namespace: default
EOF`,
            },
            {
              label: "Step 4: Deploy DaemonSet",
              code: `kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: icp-agent
  namespace: default
  labels:
    app: icp-agent
spec:
  selector:
    matchLabels:
      app: icp-agent
  template:
    metadata:
      labels:
        app: icp-agent
    spec:
      serviceAccountName: icp-agent
      hostPID: true
      hostNetwork: false

      initContainers:
        - name: init-socket-dir
          image: busybox:1.36
          command:
            - sh
            - -c
            - |
              mkdir -p /run/spire/sockets
              chmod 0777 /run/spire/sockets
          volumeMounts:
            - name: spire-agent-socket-dir
              mountPath: /run/spire/sockets

      containers:
        - name: icp-agent
          image: your-docker-registry.example.com/icp-agent:latest
          imagePullPolicy: Always

          command:
            - "icp-agent"
            - "-c"
            - "/etc/icp-agent/config.yaml"

          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name

          securityContext:
            runAsUser: 0
            runAsGroup: 0
            runAsNonRoot: false
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
              add:
                - SYS_PTRACE
            seccompProfile:
              type: RuntimeDefault

          volumeMounts:
            - name: config
              mountPath: /etc/icp-agent
              readOnly: true
            - name: spire-agent-socket-dir
              mountPath: /run/spire/sockets
              readOnly: false
            - name: agent-data
              mountPath: /var/lib/icp-agent
              readOnly: false
            - name: agent-tmp
              mountPath: /tmp
              readOnly: false
            - name: proc
              mountPath: /proc
              readOnly: true
            - name: sa-token
              mountPath: /var/run/secrets/kubernetes.io/serviceaccount
              readOnly: true

          resources:
            limits:
              cpu: "500m"
              memory: "512Mi"
            requests:
              cpu: "100m"
              memory: "128Mi"

          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 60

          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30

      volumes:
        - name: config
          configMap:
            name: icp-agent-config
        - name: spire-agent-socket-dir
          hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
        - name: agent-data
          emptyDir:
            sizeLimit: 1Gi
        - name: agent-tmp
          emptyDir:
            sizeLimit: 512Mi
        - name: proc
          hostPath:
            path: /proc
            type: Directory
        - name: sa-token
          projected:
            sources:
              - serviceAccountToken:
                  path: token
                  expirationSeconds: 3600

      tolerations:
        - operator: Exists

      dnsPolicy: ClusterFirst
EOF`,
            },
          ],
        },
        {
          methodName: "Workload Deployment",
          steps: [
            {
              label: "Agent SPIFFE ID format:",
              code: `spiffe://<tenant-id>/agent/<node-name>`,
            },
            {
              label: "Finding Your Workload's Node",
              code: `# Deploy your workload first
kubectl apply -f your-workload.yaml

# Find which node it's running on
kubectl get pods -n default -o wide

# Example output:
# NAME                       READY   STATUS    NODE
# my-app-7984bc7b57-9xsk4    1/1     Running   k8s-node-01`,
            },
            {
              label:
                "Workload Deployment Example(File: my-app-deployment.yaml)",
              code: `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  labels:
    app: my-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
        version: v1
        environment: production
    spec:
      serviceAccountName: my-app
      nodeSelector:
        kubernetes.io/hostname: node-name # Must match the registered Parent Agent ID
      containers:
      - name: my-app
        image: my-registry.example.com/my-app:latest
        ports:
        - containerPort: 8080

        env:
        # CRITICAL: SPIFFE socket path
        - name: SPIFFE_ENDPOINT_SOCKET
          value: "unix:///run/spire/sockets/agent.sock"

        # CRITICAL: Kubernetes Downward API for attestation
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_UID
          valueFrom:
            fieldRef:
              fieldPath: metadata.uid
        - name: SERVICE_ACCOUNT
          valueFrom:
            fieldRef:
              fieldPath: spec.serviceAccountName
        - name: POD_LABEL_APP
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['app']

        # Application config
        - name: LOG_LEVEL
          value: "info"

        volumeMounts:
        # CRITICAL: Mount agent socket
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
          readOnly: true

        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"

        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30

        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10

      volumes:
      # CRITICAL: hostPath volume for agent socket
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: Directory
---
apiVersion: v1
kind: Service
metadata:
  name: my-app
  namespace: default
spec:
  selector:
    app: my-app
  ports:
  - name: http
    protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP`,
            },
            {
              label: "Deploy:",
              code: `kubectl apply -f my-app-deployment.yaml`,
            },
          ],
        },
      ],
      typescript: [],
    },
  },
  {
    id: "2",
    question: "How do I deploy my agent on docker?",
    description: "Learn how to deploy SPIRE agents using Helm, manual setup",
    code: {
      python: [
        {
          methodName: "Quick Start with Docker Compose",
          steps: [
            {
              label: "Step 1: Create docker-compose.yml",
              code: `mkdir icp-demo
cd icp-demo

cat > docker-compose.yml <<'EOF'
version: '3.8'

services:
  # ICP Agent
  icp-agent:
    image: your-docker-registry.example.com/icp-agent:latest
    container_name: icp-agent
    hostname: icp-agent-docker

    environment:
      # Tenant configuration
      - ICP_AGENT_AGENT__TENANT_ID=your-tenant-id-here
      - ICP_AGENT_AGENT__NODE_ID=docker-prod-host-01

      # ICP Server connection
      - ICP_AGENT_ICP_SERVICE__ADDRESS=https://stage.api.authsec.dev/spiresvc

      # Attestation
      - ICP_AGENT_ATTESTATION__TYPE=auto

      # Logging
      - ICP_AGENT_LOGGING__LEVEL=info
      - ICP_AGENT_LOGGING__FORMAT=json

    volumes:
      # Shared socket for workloads
      - agent-socket:/run/spire/sockets

      # Docker API access (for container attestation)
      - /var/run/docker.sock:/var/run/docker.sock:ro

    networks:
      - icp-network

    restart: unless-stopped

    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

  # Example Workload A (Web Service)
  web-service:
    build: ./web-service
    container_name: web-service
    hostname: web-service

    depends_on:
      icp-agent:
        condition: service_healthy

    environment:
      # SPIFFE socket
      - SPIFFE_ENDPOINT_SOCKET=unix:///run/spire/sockets/agent.sock

      # Docker metadata (for attestation)
      - DOCKER_CONTAINER_ID=$\{HOSTNAME}
      - DOCKER_CONTAINER_NAME=web-service
      - DOCKER_IMAGE_NAME=web-service:latest

      # Application config
      - PORT=8080
      - LOG_LEVEL=info

    volumes:
      # Mount agent socket
      - agent-socket:/run/spire/sockets:ro

    networks:
      - icp-network

    ports:
      - "8080:8080"

    labels:
      # CRITICAL: These labels are used for attestation
      - "app=web-service"
      - "env=production"
      - "version=v1"

    restart: unless-stopped

  # Example Workload B (API Service)
  api-service:
    build: ./api-service
    container_name: api-service
    hostname: api-service

    depends_on:
      icp-agent:
        condition: service_healthy

    environment:
      - SPIFFE_ENDPOINT_SOCKET=unix:///run/spire/sockets/agent.sock
      - DOCKER_CONTAINER_ID=$\{HOSTNAME}
      - DOCKER_CONTAINER_NAME=api-service
      - DOCKER_IMAGE_NAME=api-service:latest
      - PORT=8443

    volumes:
      - agent-socket:/run/spire/sockets:ro

    networks:
      - icp-network

    ports:
      - "8443:8443"

    labels:
      - "app=api-service"
      - "env=production"
      - "version=v1"

    restart: unless-stopped

volumes:
  agent-socket:
    driver: local

networks:
  icp-network:
    driver: bridge
EOF`,
            },
            {
              label: "Step 2: Create Example Workload(Directory Structure)",
              code: `icp-demo/
├── docker-compose.yml
├── web-service/
│   ├── Dockerfile
│   ├── app.py
│   └── requirements.txt
└── api-service/
    ├── Dockerfile
    ├── app.py
    └── requirements.txt`,
            },
            {
              label: "File: web-service/requirements.txt",
              code: `git+https://github.com/authsec-ai/sdk-authsec.git
fastapi
uvicorn
httpx`,
            },
            {
              label: "File: web-service/app.py",
              code: `import asyncio
from authsec_sdk import QuickStartSVID
from fastapi import FastAPI
import uvicorn
import httpx

app = FastAPI()
svid = None

@app.on_event("startup")
async def startup():
    global svid
    svid = await QuickStartSVID.initialize()
    print(f"✅ Web Service authenticated as: {svid.spiffe_id}")

@app.get("/healthz")
async def health():
    return {"status": "healthy"}

@app.get("/")
async def root():
    return {
        "service": "web-service",
        "spiffe_id": svid.spiffe_id if svid else None
    }

@app.get("/call-api")
async def call_api():
    """Call API service with mTLS"""
    ssl_context = svid.create_ssl_context_for_client()

    async with httpx.AsyncClient(verify=ssl_context) as client:
        response = await client.get("https://api-service:8443/api/data")
        return response.json()

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)`,
            },
            {
              label: "File: web-service/Dockerfile",
              code: `FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY app.py .

# Run as non-root user
RUN useradd -m -u 1000 appuser
USER appuser

EXPOSE 8080

CMD ["python", "app.py"]`,
            },
            {
              label: "Step 3: Start Services",
              code: `# Start all services
docker compose up -d

# Check status
docker compose ps

# Check logs
docker compose logs -f icp-agent

# Verify agent health
curl http://localhost:8080/healthz`,
            },
            {
              label: "Expected output:",
              code: `{"status":"healthy"}`,
            },
            {
              label: "Step 4: Register Workloads",
              code: `# Set variables
export ICP_SERVER_URL="https://stage.api.authsec.dev/spiresvc"
export TENANT_ID="your-tenant-id-here"
export NODE_ID="docker-prod-host-01"

# Register web-service
curl -X POST "$\{ICP_SERVER_URL}/api/v1/workloads" \
  -H "Content-Type: application/json" \
  -d '{
    "spiffe_id": "spiffe://your-spiffe-id",
    "parent_id": "spiffe://your-parent-id",
    "selectors": {
      "docker:label:app": "web-service",
      "docker:label:env": "production"
    },
    "ttl": 3600
  }'

# Register api-service
curl -X POST "$\{ICP_SERVER_URL}/api/v1/workloads" \
  -H "Content-Type: application/json" \
  -d '{
    "spiffe_id": "spiffe://your-spiffe-id",
    "parent_id": "spiffe://your-parent-id",
    "selectors": {
      "docker:label:app": "api-service",
      "docker:label:env": "production"
    },
    "ttl": 3600
  }'`,
            },
            {
              label: "Step 5: Test",
              code: `# Test web service
curl http://localhost:8080/

# Expected output:
# {
#   "service": "web-service",
#   "spiffe_id": "spiffe://your-trust-domain.example.com/workload/web-service"
# }

# Test mTLS communication
curl http://localhost:8080/call-api`,
            },
          ],
        },
        {
          methodName: "Workload Registration",
          steps: [
            {
              label: "Workload Registration UI",
              image: "/workload-img3.png",
              imageAlt: "Workload Registration Interface Screenshot",
            },
            {
              label: "Guide: Registering Workloads(Docker)",
              code: `1. Give Workload Name
2. Select Parent Agent (e.g., icp-agent-docker)
3. Select Platform: Docker
4. Give contaier label values (e.g., app=web-service, env=production)
5. Add selector(optional)
6. Click Register Workload`,
            },
          ],
        },
      ],
      typescript: [],
    },
  },
  {
    id: "3",
    question: "How do I deploy my agent on VM(Virtual Machine)?",
    description: "Learn how to deploy SPIRE agents using Helm, manual setup",
    code: {
      python: [
        {
          methodName: "Method 1: Quick Install Script",
          steps: [
            {
              label: "Installation",
              code: `# Download and run installer
curl -fsSL https://install.authsec.ai/icp-agent.sh | sudo bash -s -- \
  --tenant-id "your-tenant-id-here" \
  --icp-server "https://stage.api.authsec.dev/spiresvc" \
  --node-id "vm-prod-web-01"`,
            },
            {
              label: "The script will:",
              code: `1. Install dependencies (Python 3, systemd)
2. Download ICP Agent binary
3. Create systemd service
4. Start the agent
5. Enable auto-start on boot`,
            },
          ],
        },
        {
          methodName: "Method 2: Manual Installation",
          steps: [
            {
              label: "Step 1: Install Dependencies(Ubuntu/Debian)",
              code: `sudo apt-get update
sudo apt-get install -y python3 python3-pip git systemd`,
            },
            {
              label: "RHEL/CentOS",
              code: `sudo yum install -y python3 python3-pip git systemd`,
            },
            {
              label: "Step 2: Download ICP Agent",
              code: `# Create installation directory
sudo mkdir -p /opt/icp-agent
cd /opt/icp-agent

# Clone repository
sudo git clone https://github.com/your-org/icp-agent.git .

# Install Python dependencies
sudo pip3 install -r requirements.txt`,
            },
            {
              label: "Step 3: Create Configuration",
              code: `# Create config directory
sudo mkdir -p /etc/icp-agent

# Create config file
sudo tee /etc/icp-agent/config.yaml > /dev/null <<EOF
agent:
  tenant_id: "your-tenant-id-here"
  node_id: "vm-prod-web-01"
  data_dir: "/var/lib/icp-agent"
  socket_path: "/run/spire/sockets/agent.sock"
  renewal_threshold: "6h"

icp_service:
  address: "https://your-icp-server.example.com/spiresvc"
  trust_bundle_path: "/etc/icp-agent/ca-bundle.pem"
  timeout: 30
  max_retries: 3
  retry_backoff: 5

attestation:
  type: "unix"
  unix:
    method: "procfs"

security:
  cache_encryption_key: ""
  cache_path: "/var/lib/icp-agent/cache/svid.cache"

logging:
  level: "info"
  format: "json"
  file_path: "/var/log/icp-agent/agent.log"

health:
  enabled: true
  port: 8080
  bind_address: "127.0.0.1"
EOF`,
            },
            {
              label: "Step 4: Create systemd Service",
              code: `sudo tee /etc/systemd/system/icp-agent.service > /dev/null <<EOF
[Unit]
Description=ICP Agent - Workload Identity Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/opt/icp-agent/icp_agent/main.py -c /etc/icp-agent/config.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/icp-agent /var/log/icp-agent /run/spire

# Environment
Environment="PYTHONUNBUFFERED=1"

[Install]
WantedBy=multi-user.target
EOF`,
            },
            {
              label: "Step 5: Create Required Directories",
              code: `# Create data and log directories
sudo mkdir -p /var/lib/icp-agent/cache
sudo mkdir -p /var/log/icp-agent
sudo mkdir -p /run/spire/sockets

# Set permissions
sudo chmod 755 /run/spire/sockets
sudo chmod 700 /var/lib/icp-agent`,
            },
            {
              label: "Step 6: Start Agent",
              code: `# Reload systemd
sudo systemctl daemon-reload

# Start agent
sudo systemctl start icp-agent

# Enable auto-start on boot
sudo systemctl enable icp-agent

# Check status
sudo systemctl status icp-agent`,
            },
            {
              label: "Verification",
              code: `# Check agent is running
sudo systemctl status icp-agent

# Check logs
sudo journalctl -u icp-agent -f

# Check socket exists
ls -l /run/spire/sockets/agent.sock

# Test health endpoint
curl http://localhost:8080/healthz`,
            },
            {
              label: "Expected output",
              code: `{"status": "healthy"}`,
            },
          ],
        },
        // {
        //   methodName: "Method 2: kubectl (Manual Deployment)",
        //   steps: [
        //     {
        //       label: "Step 1: Create Namespace",
        //       code: `kubectl create namespace default`,
        //     },
        //   ],
        // },
      ],
      typescript: [],
    },
  },
];

// CodeBlock component for displaying code snippets or images
const CodeBlock = ({
  code,
  label,
  onCopy,
  copied,
  image,
  imageAlt,
}: {
  code?: string;
  label?: string;
  onCopy?: () => void;
  copied?: boolean;
  image?: string;
  imageAlt?: string;
}) => {
  return (
    <div className="border border-neutral-700 rounded-lg overflow-hidden bg-neutral-800 backdrop-blur-sm">
      <div className="flex items-center justify-between border-b border-neutral-700 bg-neutral-800 px-2.5 py-0.5">
        <span className="text-[11px] font-semibold text-neutral-400">
          {label}
        </span>
        <div className="flex items-center">
          {onCopy && code && (
            <Button
              size="sm"
              variant="ghost"
              className="h-5 w-5 p-0 hover:bg-neutral-700 text-neutral-400 hover:text-neutral-300"
              onClick={onCopy}
            >
              {copied ? (
                <Check className="h-2.5 w-2.5 text-green-500" />
              ) : (
                <Copy className="h-2.5 w-2.5" />
              )}
            </Button>
          )}
        </div>
      </div>
      <div className="p-3 overflow-x-auto-hidden bg-neutral-900">
        {image ? (
          <img
            src={image}
            alt={imageAlt || label || "Screenshot"}
            className="w-full h-auto rounded border border-neutral-700"
          />
        ) : (
          <pre className="text-sm font-mono text-neutral-300 whitespace-pre-wrap leading-relaxed">
            {code}
          </pre>
        )}
      </div>
    </div>
  );
};

// Agent FAQ Component
function AgentFAQ() {
  const [isOpen, setIsOpen] = useState(false);
  const [showTooltip, setShowTooltip] = useState(false);
  const [selectedFAQ, setSelectedFAQ] = useState<FAQItem | null>(null);
  const [copiedLanguage, setCopiedLanguage] = useState<string | null>(null);
  const [activeSubTab, setActiveSubTab] = useState<number>(0);

  const handleCopy = (code: string, id: string) => {
    navigator.clipboard.writeText(code);
    setCopiedLanguage(id);
    setTimeout(() => setCopiedLanguage(null), 2000);
  };

  const toggleFAQ = () => {
    setIsOpen(!isOpen);
    setShowTooltip(false);
  };

  const handleQuestionClick = (faq: FAQItem) => {
    setSelectedFAQ(faq);
    setCopiedLanguage(null);
    setActiveSubTab(0);
  };

  const handleCloseModal = () => {
    setSelectedFAQ(null);
    setCopiedLanguage(null);
  };

  return (
    <>
      <div
        className={cn(
          "fixed bottom-6 right-6 z-40 flex gap-2",
          isOpen ? "items-end" : "items-center"
        )}
      >
        {showTooltip && !isOpen && (
          <div className="bg-slate-900 dark:bg-neutral-800 text-white px-3 py-2 rounded-lg shadow-lg animate-in fade-in duration-200">
            <span className="text-sm font-medium whitespace-nowrap">
              Quick Help
            </span>
          </div>
        )}

        {isOpen && (
          <div className="p-4 w-80 animate-in slide-in-from-right-2 fade-in duration-200">
            <div className="space-y-2">
              {SPIRE_FAQ_DATA.map((faq) => (
                <button
                  key={faq.id}
                  onClick={() => handleQuestionClick(faq)}
                  className="w-full text-left p-3 rounded-md bg-slate-50 dark:bg-neutral-800 hover:bg-slate-100 dark:hover:bg-neutral-700 transition-colors border border-slate-200 dark:border-neutral-700"
                >
                  <div className="flex items-start gap-2">
                    <Code2 className="h-4 w-4 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                    <span className="text-sm text-foreground font-normal">
                      {faq.question}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        <Button
          className={cn(
            "h-10 w-10 rounded-full shadow-lg transition-all duration-200 flex-shrink-0",
            "bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:text-white dark:hover:bg-blue-600",
            isOpen && "rotate-180",
            "cursor-pointer"
          )}
          size="icon"
          onClick={toggleFAQ}
          onMouseEnter={() => setShowTooltip(true)}
          onMouseLeave={() => setShowTooltip(false)}
        >
          {isOpen ? (
            <X className="h-4 w-4 text-white" />
          ) : (
            <HelpCircle className="h-4 w-4 text-white" />
          )}
        </Button>
      </div>

      <Dialog
        open={!!selectedFAQ}
        onOpenChange={(open) => !open && handleCloseModal()}
      >
        <DialogContent className="!max-w-none w-[65vw] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="text-lg font-semibold">
              {selectedFAQ?.question}
            </DialogTitle>
            <DialogDescription>{selectedFAQ?.description}</DialogDescription>
          </DialogHeader>

          {selectedFAQ && selectedFAQ.code && (
            <Tabs
              value={String(activeSubTab)}
              onValueChange={(v) => setActiveSubTab(Number(v))}
              className="mt-4"
            >
              <TabsList
                className="grid w-full"
                style={{
                  gridTemplateColumns: `repeat(${selectedFAQ.code.python.length}, 1fr)`,
                }}
              >
                {selectedFAQ.code.python.map((method, idx) => (
                  <TabsTrigger key={idx} value={String(idx)}>
                    {method.methodName}
                  </TabsTrigger>
                ))}
              </TabsList>

              {selectedFAQ.code.python.map((method, methodIdx) => (
                <TabsContent
                  key={methodIdx}
                  value={String(methodIdx)}
                  className="mt-4 space-y-4"
                >
                  {method.steps.map((step, stepIdx) => (
                    <CodeBlock
                      key={stepIdx}
                      label={step.label}
                      code={step.code}
                      image={step.image}
                      imageAlt={step.imageAlt}
                      onCopy={
                        step.code
                          ? () =>
                              handleCopy(
                                step.code!,
                                `method-${methodIdx}-${stepIdx}`
                              )
                          : undefined
                      }
                      copied={
                        copiedLanguage === `method-${methodIdx}-${stepIdx}`
                      }
                    />
                  ))}
                </TabsContent>
              ))}
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

const getStatusColor = (status: string) => {
  const statusLower = status.toLowerCase();
  if (statusLower === "active" || statusLower === "healthy") {
    return "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400";
  }
  if (statusLower === "inactive" || statusLower === "unhealthy") {
    return "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400";
  }
  return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400";
};

const formatDate = (dateString: string): string => {
  if (!dateString) return "—";
  try {
    const date = new Date(dateString);
    return date.toLocaleString();
  } catch {
    return dateString;
  }
};

const formatTimeAgo = (dateString: string): string => {
  if (!dateString) return "Unknown";
  try {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;

    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;

    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  } catch {
    return "Unknown";
  }
};

export function AgentsPage() {
  const sessionData = SessionManager.getSession();
  const navigate = useNavigate();

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["spire-agents-intro"],
  });

  const {
    data: agentsData,
    isLoading: agentsLoading,
    isFetching: agentsFetching,
    error: agentsError,
    refetch,
  } = useListAgentsQuery(undefined, {
    skip: !sessionData?.token,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const agents = agentsData || [];

  // Calculate stats
  const stats: AgentStats = useMemo(() => {
    const active = agents.filter(
      (a) =>
        a.status?.toLowerCase() === "active" ||
        a.status?.toLowerCase() === "healthy"
    ).length;

    return {
      active,
      inactive: agents.length - active,
      total: agents.length,
    };
  }, [agents]);

  const showInitialSkeleton = agentsLoading && !agentsData;
  const isTableLoading = agentsFetching && agentsData;

  // const handleCancel = () => {
  //   navigate("/clients/workloads");
  // };

  return (
    <div className="min-h-screen">
      <div className="space-y-6 p-6 max-w-[1600px] mx-auto">
        <HeaderCard>
          <div className="flex justify-between items-center gap-4 p-6">
            <div className="flex items-center gap-3">
              <div>
                <h1 className="text-2xl font-semibold">SPIRE Agents</h1>
                <p className="text-sm text-foreground mt-1">
                  Monitor and manage SPIRE agent nodes in your infrastructure
                </p>
              </div>
            </div>
            <div className="flex items-center">
              <Button
                variant="outline"
                size="sm"
                onClick={() => refetch()}
                data-tour-id="agents-refresh"
              >
                <RefreshCw
                  className={`h-4 w-4 mr-2 ${
                    isTableLoading ? "animate-spin" : ""
                  }`}
                />
                Refresh
              </Button>
            </div>
          </div>
        </HeaderCard>

        {/* Agents Table */}
        <div className="agents-table-container" data-tour-id="agents-table">
          <style>{`
            .agents-table-container [data-slot="table-container"] {
              border: none !important;
              background: transparent !important;
            }
            .agents-table-container [data-slot="table-header"] {
              background: transparent !important;
            }
            .agents-table-container .bg-muted\\/50,
            .agents-table-container .bg-muted\\/30,
            .agents-table-container [class*="bg-muted"] {
              background: transparent !important;
            }
            .agents-table-container .hover\\:bg-muted\\/50:hover,
            .agents-table-container .hover\\:bg-muted\\/30:hover {
              background: rgba(148, 163, 184, 0.1) !important;
            }
            .agents-table-container .border,
            .agents-table-container .border-b {
              border-color: rgba(148, 163, 184, 0.2) !important;
            }
            .agents-table-container .shadow-xl {
              box-shadow: none !important;
            }
          `}</style>
          <TableCard className="transition-all duration-500">
            <CardContent variant="flush">
              {agentsError ? (
                <div className="p-8 text-center">
                  <div className="flex flex-col items-center space-y-4">
                    <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                      <Server className="h-8 w-8 text-red-600 dark:text-red-400" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-red-900 dark:text-red-100">
                        Unable to Load Agents
                      </h3>
                      <p className="text-red-700 dark:text-red-300 mt-1">
                        Failed to fetch agent data
                      </p>
                    </div>
                  </div>
                </div>
              ) : showInitialSkeleton ? (
                <DataTableSkeleton rows={10} />
              ) : agents.length === 0 ? (
                <div className="p-12 text-center">
                  <div className="flex flex-col items-center space-y-4">
                    <div className="p-4 bg-black/5 dark:bg-white/5 rounded-full">
                      <Server className="h-8 w-8 text-foreground/50" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-foreground">
                        No Agents Found
                      </h3>
                      <p className="text-foreground/70 mt-1">
                        No SPIRE agents are currently registered
                      </p>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="relative overflow-hidden">
                  <table className="w-full text-sm table-auto">
                    <thead>
                      <tr className="border-b border-border">
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          Agent ID
                        </th>
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          SPIFFE ID
                        </th>
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          Node ID
                        </th>
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          Attestation Type
                        </th>
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          Status
                        </th>
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          Last Seen
                        </th>
                        <th className="text-left py-4 px-6 font-semibold text-foreground">
                          Created
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {agents.map((agent) => (
                        <tr
                          key={agent.id}
                          className="border-b border-border hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
                        >
                          <td className="py-4 px-6">
                            <div className="flex items-center gap-2">
                              <Activity className="h-4 w-4 text-foreground/50" />
                              <span className="font-mono text-xs text-foreground">
                                {agent.id.substring(0, 12)}...
                              </span>
                            </div>
                          </td>
                          <td className="py-4 px-6">
                            <span className="font-mono text-xs text-foreground">
                              {agent.spiffe_id}
                            </span>
                          </td>
                          <td className="py-4 px-6">
                            <span className="font-mono text-xs text-foreground">
                              {agent.node_id || "—"}
                            </span>
                          </td>
                          <td className="py-4 px-6">
                            <Badge
                              variant="outline"
                              className="font-mono text-xs"
                            >
                              {agent.attestation_type || "unknown"}
                            </Badge>
                          </td>
                          <td className="py-4 px-6">
                            <Badge className={getStatusColor(agent.status)}>
                              {agent.status}
                            </Badge>
                          </td>
                          <td className="py-4 px-6">
                            <div className="flex flex-col">
                              <span className="text-xs text-foreground font-medium">
                                {formatTimeAgo(agent.last_seen)}
                              </span>
                              <span className="text-xs text-foreground/50">
                                {formatDate(agent.last_seen)}
                              </span>
                            </div>
                          </td>
                          <td className="py-4 px-6 text-xs text-foreground">
                            {formatDate(agent.created_at)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {isTableLoading && (
                    <div className="absolute inset-0 bg-background/50 backdrop-blur-sm flex items-center justify-center">
                      <div className="flex items-center space-x-2">
                        <RefreshCw className="h-4 w-4 animate-spin" />
                        <span className="text-sm font-medium text-foreground">
                          Refreshing...
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </TableCard>
        </div>
      </div>
      {/* <AgentFAQ /> */}
    </div>
  );
}
