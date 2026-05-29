import { DataSource } from 'typeorm';
import { User } from '../models/userModel.js';
import dotenv from 'dotenv';
import { VerificationToken } from '../models/verificationTokenModel.js';
import { Session } from '../models/sessionModel.js';
import { Account } from '../models/accountModel.js';

dotenv.config();

if (!process.env.DB_URI) {
  throw new Error('CRITICAL ERROR: DB_URI environment variable is missing.');
}

export const AppDataSource = new DataSource({
  type: 'postgres',
  url: process.env.DB_URI, 
  synchronize: true, 
  logging: false,
  entities: [User, Account, Session, VerificationToken], 
});

export const connectDB = async () => {
  try {
    await AppDataSource.initialize();
    console.log('PostgreSQL Connected via TypeORM');
  } catch (error) {
    console.error('Error connecting to PostgreSQL:', error);
    process.exit(1);
  }
};