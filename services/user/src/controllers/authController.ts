import { type Request, type Response } from 'express';
import jwt from 'jsonwebtoken';
import { AppDataSource } from '../config/db.js';
import { User } from '../models/userModel.js';

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

export const getUserProfile = async (req: Request, res: Response) => {
  try {
    const userId = (req.user as any)?.id; 

    if (!userId) {
      return res.status(401).json({ error: 'Unauthorized: No user ID found' });
    }

    const userRepository = AppDataSource.getRepository(User);
    
    // Updated to pull your specific fields
    const user = await userRepository.findOne({
      where: { id: userId },
      select: {
        id: true,
        name: true,
        email: true,
        image: true,
      }, 
    });

    if (!user) {
      return res.status(404).json({ error: 'User not found' });
    }

    // Return the exact fields
    return res.status(200).json({
      id: user.id,
      name: user.name,
      email: user.email,
      image: user.image,
    });

  } catch (error) {
    console.error('Error fetching user profile:', error);
    return res.status(500).json({ error: 'Internal Server Error' });
  }
};

export const getPublicProfile = async (req: Request, res: Response) => {
  try {
    // 1. Force TypeScript to treat this strictly as a single string
    const id = req.params.id as string; 
    
    const userRepository = AppDataSource.getRepository(User); 
    
    const user = await userRepository.findOne({
      where: { id: id },
      select: {
        id: true,
        name: true,
        image: true
      },
    });

    if (!user) {
      return res.status(404).json({ error: 'User not found in database' });
    }

    return res.status(200).json(user);
  } catch (error) {
    console.error('Profile fetch error:', error);
    return res.status(500).json({ error: 'Internal Server Error' });
  }
};