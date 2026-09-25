import { Router } from 'express';
import { generateReportPDF } from '../controllers/report.controller';

const router = Router();

router.post('/reports/sales-pdf', generateReportPDF);

export default router;

