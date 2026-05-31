import express, { type Application } from 'express';
// import cors from 'cors';
import passport from 'passport';
import authRoutes from './routes/authRoutes.js';
import userRoutes from './routes/userRoutes.js';
import './config/passport.js'; // Initialize passport config

const app: Application = express();

// Middlewares
// app.use(cors({ origin: process.env.CLIENT_URL, credentials: true }));
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(passport.initialize());

// Routes
app.use('/api/auth', authRoutes);
app.use('/api/users', userRoutes);

export default app;