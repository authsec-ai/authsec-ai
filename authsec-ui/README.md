# AuthSec UI

The frontend web application for AuthSec — an open-source enterprise Identity and Access Management (IAM) platform. Built with React 19, Vite 6, and TypeScript.

## Tech Stack

| Layer            | Technology                |
| ---------------- | ------------------------- |
| Framework        | React 19 + TypeScript 5.8 |
| Build tool       | Vite 6                    |
| State management | Redux Toolkit             |
| Routing          | React Router 6            |
| UI components    | Radix UI                  |
| Styling          | Tailwind CSS 4            |
| Forms            | React Hook Form + Zod     |
| Data fetching    | RTK Query                 |

## Requirements

- **Node.js** >= 20.0.0
- **npm** >= 10.0.0
- A running **AuthSec backend** instance (see backend repository)

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/authsec-ai/Authsec-ui.git
cd Authsec-ui
```

### 2. Install dependencies

```bash
npm install
```

### 3. Configure environment

```bash
cp .env.example .env
```

Edit `.env` and set the URL of your AuthSec backend:

```env
VITE_API_URL=http://localhost:3000
VITE_APP_NAME=AuthSec Enterprise IAM
```

### 4. Start the development server

```bash
npm run dev
```

The app will be available at [http://localhost:5173](http://localhost:5173).

## Environment Variables

| Variable        | Required | Description                                                          |
| --------------- | -------- | -------------------------------------------------------------------- |
| `VITE_API_URL`  | Yes      | URL of the AuthSec backend API                                       |
| `VITE_APP_NAME` | No       | Application name shown in the UI (default: `AuthSec Enterprise IAM`) |

## Available Scripts

| Script                  | Description                            |
| ----------------------- | -------------------------------------- |
| `npm run dev`           | Start development server               |
| `npm run build`         | Build for production (output: `dist/`) |
| `npm run preview`       | Preview the production build locally   |
| `npm run lint`          | Run ESLint                             |
| `npm run lint:fix`      | Auto-fix lint issues                   |
| `npm run type-check`    | Run TypeScript type checking           |
| `npm run test`          | Run tests with Vitest                  |
| `npm run test:coverage` | Run tests with coverage report         |

## Production Build

```bash
npm run build
```

The built files will be in the `dist/` directory. Serve them with any static file server (nginx, Caddy, etc.) or deploy to a CDN.

## Hosted Platform

If you'd prefer not to self-host, AuthSec is available as a managed service at **[app.authsec.ai](https://app.authsec.ai)**. Sign up there to get a fully managed backend without any infrastructure setup.

## Backend

This frontend requires the AuthSec backend to be running and accessible. The backend provides the API endpoints for authentication, user management, client configuration, and all IAM features.

Set `VITE_API_URL` to point to your backend instance.

## License

[Apache 2.0](LICENSE)
