import { Request, Response } from 'express';

const startTime = new Date();

export const getHealth = (_req: Request, res: Response): void => {
  res.status(200).json({
    status: 'UP',
    service: 'kather_baksho-worker-ts',
    version: '1.0.0',
    uptimeSeconds: Math.floor((Date.now() - startTime.getTime()) / 1000),
    timestamp: new Date().toISOString(),
    nodeVersion: process.version,
    memoryUsageMB: Math.round(process.memoryUsage().heapUsed / 1024 / 1024),
    features: ['pdf-invoice-generation', 'sales-report-generation', 'event-worker']
  });
};

export const getReadiness = (_req: Request, res: Response): void => {
  res.status(200).json({
    status: 'ready',
    service: 'kather_baksho-worker-ts',
    checks: {
      memory: 'healthy',
      pdfEngine: 'ready'
    }
  });
};

