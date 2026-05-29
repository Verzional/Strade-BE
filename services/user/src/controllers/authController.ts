import { type Request, type Response } from 'express';
import jwt from 'jsonwebtoken';
import { type User } from '../models/userModel.js';

export const googleCallback = (req: Request, res: Response) => {
  try {
    const user = req.user as User; 
    
    if (!user) {
      return res.redirect(`${process.env.CLIENT_URL}/login?error=AuthenticationFailed`);
    }

    const token = jwt.sign(
      { id: user.id, email: user.email }, 
      process.env.JWT_SECRET as string, 
      { expiresIn: '7d' }
    );

    res.redirect(`${process.env.CLIENT_URL}/auth/success?token=${token}`);
  } catch (error) {
    console.error('Auth Callback Error:', error);
    res.redirect(`${process.env.CLIENT_URL}/login?error=ServerError`);
  }
};