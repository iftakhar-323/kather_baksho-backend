import express from 'express';
import cors from 'cors';
import dotenv from 'dotenv';
import healthRoutes from './routes/health.routes';
import invoiceRoutes from './routes/invoice.routes';
import reportRoutes from './routes/report.routes';

dotenv.config();

const app = express();
const PORT = process.env.PORT || 8083;

// Middlewares
app.use(cors({ origin: '*' }));
app.use(express.json({ limit: '10mb' }));

// Structured Request Logging
app.use((req, res, next) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(
      JSON.stringify({
        timestamp: new Date().toISOString(),
        service: 'kather_baksho-worker-ts',
        method: req.method,
        path: req.path,
        status: res.statusCode,
        duration_ms: duration,
        ip: req.ip || req.socket.remoteAddress
      })
    );
  });
  next();
});

// Root metadata route
app.get('/', (_req, res) => {
  res.json({
    service: 'kather_baksho-worker-ts',
    description: 'Enterprise Node.js & TypeScript Microservice for PDF Invoices & Async Reports',
    endpoints: {
      health: 'GET /health',
      invoice: 'POST /api/v1/invoices/generate',
      report: 'POST /api/v1/reports/sales-pdf'
    }
  });
});

// Mount Routes
app.use(healthRoutes);
app.use('/api/v1', invoiceRoutes);
app.use('/api/v1', reportRoutes);

// 404 Handler
app.use((req, res) => {
  res.status(404).json({ error: `Path not found: ${req.method} ${req.path}` });
});

// Server Listen
const server = app.listen(PORT, () => {
  console.log(`[Worker-TS] Node.js & TypeScript Microservice running on port ${PORT}`);
});

// Graceful Shutdown
process.on('SIGTERM', () => {
  console.log('[Worker-TS] SIGTERM received. Shutting down gracefully...');
  server.close(() => {
    console.log('[Worker-TS] Process terminated.');
    process.exit(0);
  });
});

export default app;

