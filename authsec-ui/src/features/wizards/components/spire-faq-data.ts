interface DeploymentMethod {
  methodName: string;
  steps: Array<{
    label: string;
    code?: string;
    image?: string;
    imageAlt?: string;
  }>;
}

interface CodeExample {
  python: DeploymentMethod[];
  typescript: DeploymentMethod[];
}

export interface FAQItem {
  id: string;
  question: string;
  description: string;
  code?: CodeExample;
  customContent?: React.ReactNode;
}

export const SPIRE_FAQ_DATA: FAQItem[] = [
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
helm install icp-agent authsec/icp-agent \\
  -f icp-agent-values.yaml \\
  --namespace default \\
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
      address: "https://test.api.authsec.dev/spiresvc"
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
      - ICP_AGENT_ICP_SERVICE__ADDRESS=https://test.api.authsec.dev/spiresvc

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
      - DOCKER_CONTAINER_ID=\${HOSTNAME}
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
      - DOCKER_CONTAINER_ID=\${HOSTNAME}
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
export ICP_SERVER_URL="https://test.api.authsec.dev/spiresvc"
export TENANT_ID="your-tenant-id-here"
export NODE_ID="docker-prod-host-01"

# Register web-service
curl -X POST "\${ICP_SERVER_URL}/api/v1/workloads" \\
  -H "Content-Type: application/json" \\
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
curl -X POST "\${ICP_SERVER_URL}/api/v1/workloads" \\
  -H "Content-Type: application/json" \\
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
curl -fsSL https://install.authsec.ai/icp-agent.sh | sudo bash -s -- \\
  --tenant-id "your-tenant-id-here" \\
  --icp-server "https://test.api.authsec.dev/spiresvc" \\
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
      ],
      typescript: [],
    },
  },
];
