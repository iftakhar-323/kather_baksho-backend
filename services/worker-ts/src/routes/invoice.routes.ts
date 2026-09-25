import { Router } from 'express';
import { generateInvoicePDF } from '../controllers/invoice.controller';

const router = Router();

router.post('/invoices/generate', generateInvoicePDF);

export default router;

