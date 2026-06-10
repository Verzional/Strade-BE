import passport from 'passport';
import { Strategy as GoogleStrategy, type Profile, type VerifyCallback } from 'passport-google-oauth20';
import { AppDataSource } from './db.js';
import { User } from '../models/userModel.js';
import { Account } from '../models/accountModel.js';
import dotenv from 'dotenv';

dotenv.config();

passport.use(
  new GoogleStrategy(
    {
      clientID: process.env.GOOGLE_CLIENT_ID as string,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET as string,
      callbackURL: 'http://localhost:8080/api/auth/google/callback',
      proxy: true,
    },
    async (accessToken: string, refreshToken: string, profile: Profile, done: VerifyCallback) => {
      try {
        const userRepository = AppDataSource.getRepository(User);
        const accountRepository = AppDataSource.getRepository(Account);

        const email = profile.emails?.[0]?.value;
        if (!email) {
          return done(new Error('No email found in Google profile'), undefined);
        }

        const existingAccount = await accountRepository.findOne({
          where: { provider: 'google', providerAccountId: profile.id },
          relations: { user: true }, 
        });

        if (existingAccount && existingAccount.user) {
          return done(null, existingAccount.user);
        }

        let user = await userRepository.findOneBy({ email });

        if (!user) {
          const newUserInput: { name: string; email: string; image?: string } = {
            name: profile.displayName,
            email: email,
          };

          const imageUrl = profile.photos?.[0]?.value;
          if (imageUrl) {
            newUserInput.image = imageUrl;
          }

          user = userRepository.create(newUserInput);
          await userRepository.save(user);
        }

        const newAccount = accountRepository.create({
          userId: user.id,
          type: 'oauth',
          provider: 'google',
          providerAccountId: profile.id,
          access_token: accessToken,
          refresh_token: refreshToken,
        });
        await accountRepository.save(newAccount);

        return done(null, user);
      } catch (error: any) {
        return done(error, undefined);
      }
    }
  )
);

export default passport;