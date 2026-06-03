// src/routes/userRoutes.ts
import { Router } from 'express';
import { getPublicProfile, getUserProfile } from '../controllers/authController.js';
import { verifyToken } from '../middlewares/authMiddleware.js';

const router: Router = Router();

// Notice how the middleware is applied here!
router.get('/profile', verifyToken, getUserProfile);
router.get('/:id', getPublicProfile);

export default router;