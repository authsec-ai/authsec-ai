const express = require('express');
const path = require('path');
const app = express();
const port = process.env.PORT || 3000;

// Middleware to log all incoming requests
app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${req.method} ${req.url} - Request received`);
  next();
});

// Dynamic config endpoint
app.get('/config.js', (req, res) => {
  const config = {
    VITE_API_URL: process.env.VITE_API_URL || 'http://localhost:3000',
    VITE_AMPLITUDE_API_KEY: process.env.VITE_AMPLITUDE_API_KEY || '',
    VITE_CLARITY_PROJECT_ID: process.env.VITE_CLARITY_PROJECT_ID || '',
    VITE_HUBSPOT_ACCESS_TOKEN: process.env.VITE_HUBSPOT_ACCESS_TOKEN || ''
  };

  const configScript = `window.ENV = ${JSON.stringify(config)};`;

  res.setHeader('Content-Type', 'application/javascript');
  res.setHeader('Cache-Control', 'no-store, no-cache, must-revalidate');
  res.send(configScript);

  console.log(`[${new Date().toISOString()}] Config served:`, config);
});

// Serve static files from the React app's dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
  const healthStatus = {
    status: 'healthy',
    uptime: process.uptime(),
    timestamp: new Date().toISOString(),
    environment: {
      VITE_API_URL: process.env.VITE_API_URL || 'not set',
      VITE_AMPLITUDE_API_KEY: process.env.VITE_AMPLITUDE_API_KEY ? '***configured***' : 'not set',
      VITE_CLARITY_PROJECT_ID: process.env.VITE_CLARITY_PROJECT_ID ? '***configured***' : 'not set',
      VITE_HUBSPOT_ACCESS_TOKEN: process.env.VITE_HUBSPOT_ACCESS_TOKEN ? '***configured***' : 'not set'
    }
  };
  console.log(`[${healthStatus.timestamp}] Health check - VITE_API_URL: ${healthStatus.environment.VITE_API_URL}, Amplitude: ${healthStatus.environment.VITE_AMPLITUDE_API_KEY}, Clarity: ${healthStatus.environment.VITE_CLARITY_PROJECT_ID}, HubSpot: ${healthStatus.environment.VITE_HUBSPOT_ACCESS_TOKEN}`);
  res.status(200).json(healthStatus);
});

// Catch-all route to serve React app
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

// Start the server
app.listen(port, () => {
  console.log(`[${new Date().toISOString()}] Server running on port ${port}`);
  console.log(`[${new Date().toISOString()}] VITE_API_URL: ${process.env.VITE_API_URL || 'not set'}`);
});