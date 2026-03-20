import {
  DEFAULT_HELP_LANGUAGE_TABS,
  FloatingHelp,
  type FloatingHelpItem,
} from "@/components/shared/FloatingHelp";
import { SPIRE_FAQ_DATA as AGENT_DEPLOYMENT_DATA } from "@/features/wizards/components/spire-faq-data";

const FAQ_DATA: FloatingHelpItem[] = [
  {
    id: "1",
    question:
      "How to Integrate authentication into your mcp server / sdk agent?",
    description:
      "Learn how to integrate your AI agent by securely authenticating your MCP server or agent using our authentication API.",
    code: {
      python: [
        {
          label: "Step 1: Install AuthSec SDK",
          code: `pip install git+https://github.com/authsec-ai/sdk-authsec.git`,
        },
        {
          label: "Step 2: Create Your Secure MCP Server (server.py)",
          code: `from authsec_sdk import protected_by_AuthSec, run_mcp_server_with_oauth

# Tool 1: Accessible to all authenticated users
@protected_by_AuthSec("hello")
async def hello(arguments: dict) -> list:
    return [{
        "type": "text",
        "text": f"Hello, {arguments['_user_info']['email']}!"
    }]

# Start the server
if __name__ == "__main__":
    run_mcp_server_with_oauth(
        client_id="your-client-id-here",
        app_name="My Secure MCP Server"
    )`,
        },
        {
          label: "Step 3: Run Your Server",
          code: `python server.py`,
        },
      ],
      typescript: [
        {
          label: "Step 1: Install Dependencies",
          code: `npm install axios`,
        },
        {
          label: "Step 2: Authentication Setup (auth.ts)",
          code: `import axios from 'axios';

interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

async function authenticateClient(
  clientId: string,
  clientSecret: string
): Promise<string | null> {
  const url = 'https://api.authsec.dev/auth/token';

  try {
    const response = await axios.post<TokenResponse>(url, {
      client_id: clientId,
      client_secret: clientSecret,
      grant_type: 'client_credentials'
    });

    const token = response.data.access_token;
    console.log(\`Successfully authenticated: \${token}\`);
    return token;
  } catch (error) {
    console.error('Authentication failed:', error);
    return null;
  }
}`,
        },
        {
          label: "Step 3: Usage",
          code: `const token = await authenticateClient('your-client-id', 'your-client-secret');`,
        },
      ],
    },
  },
  {
    id: "2",
    question: "How to delegate trust to a workload? (coming soon)",
    description:
      "Learn how to integrate your AI agent by securely authenticating your MCP server or agent using our authentication API.",
    disabled: true,
    code: {
      python: [
        {
          label: "Step 1: Install AuthSec SDK",
          code: `pip install git+https://github.com/authsec-ai/sdk-authsec.git`,
        },
        {
          label: "Step 2: Create Your Secure MCP Server (server.py)",
          code: `from authsec_sdk import protected_by_AuthSec, run_mcp_server_with_oauth

# Tool 1: Accessible to all authenticated users
@protected_by_AuthSec("hello")
async def hello(arguments: dict) -> list:
    return [{
        "type": "text",
        "text": f"Hello, {arguments['_user_info']['email']}!"
    }]

# Start the server
if __name__ == "__main__":
    run_mcp_server_with_oauth(
        client_id="your-client-id-here",
        app_name="My Secure MCP Server"
    )`,
        },
        {
          label: "Step 3: Run Your Server",
          code: `python server.py`,
        },
      ],
      typescript: [
        {
          label: "Step 1: Install Dependencies",
          code: `npm install axios`,
        },
        {
          label: "Step 2: Authentication Setup (auth.ts)",
          code: `import axios from 'axios';

interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

async function authenticateClient(
  clientId: string,
  clientSecret: string
): Promise<string | null> {
  const url = 'https://api.authsec.dev/auth/token';

  try {
    const response = await axios.post<TokenResponse>(url, {
      client_id: clientId,
      client_secret: clientSecret,
      grant_type: 'client_credentials'
    });

    const token = response.data.access_token;
    console.log(\`Successfully authenticated: \${token}\`);
    return token;
  } catch (error) {
    console.error('Authentication failed:', error);
    return null;
  }
}`,
        },
        {
          label: "Step 3: Usage",
          code: `const token = await authenticateClient('your-client-id', 'your-client-secret');`,
        },
      ],
    },
  },
  {
    id: "3",
    question:
      "How do you do autonomous agent (machine to machine) authorization?",
    description:
      "Learn how to deploy SPIRE agents on Kubernetes, Docker, and VM environments",
    docsLink:
      "https://docs.authsec.dev/m2m-auth/m2m-auth-05-quick-start-guide/",
    languageTabs: [
      { key: "kubernetes", label: "Kubernetes" },
      { key: "docker", label: "Docker" },
      { key: "vm", label: "VM" },
    ],
    code: {
      kubernetes: AGENT_DEPLOYMENT_DATA[0].code?.python || [],
      docker: AGENT_DEPLOYMENT_DATA[1].code?.python || [],
      vm: AGENT_DEPLOYMENT_DATA[2].code?.python || [],
    },
  },
];

interface FloatingFAQProps {
  faqData?: FloatingHelpItem[];
}

export function FloatingFAQ({ faqData = FAQ_DATA }: FloatingFAQProps) {
  return (
    <FloatingHelp
      items={faqData}
      tooltipLabel="Quick Help"
      defaultOpen={false}
      defaultLanguage="python"
      languageTabs={DEFAULT_HELP_LANGUAGE_TABS}
      visualVariant="editorial"
    />
  );
}
