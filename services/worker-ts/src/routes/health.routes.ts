import { Router } from 'express';
import { getHealth, getReadiness } from '../controllers/health.controller';

const router = Router();

router.get('/health', getHealth);
router.get('/health/ready', getReadiness);
router.get('/api/health', getHealth);

export default router;

