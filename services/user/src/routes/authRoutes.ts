import { Router } from 'express';
import passport from 'passport';
import { googleCallback } from '../controllers/authController.js';

const router: Router = Router();

// Route to initiate Google OAuth
router.get(
  '/google',
  passport.authenticate('google', { scope: ['profile', 'email'], session: false })
);

// Google OAuth callback route
router.get(
  '/google/callback',
  passport.authenticate('google', { session: false, failureRedirect: '/login' }),
  googleCallback
);

export default router;