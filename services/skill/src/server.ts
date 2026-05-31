import "reflect-metadata";
import app from "./app.js";
import { connectDB } from "./config/db.js";
import dotenv from "dotenv";

dotenv.config();

const PORT = process.env.PORT || 3002;

const startServer = async () => {
  await connectDB();
  app.listen(PORT, () => {
    console.log(`Skill Service running on port ${PORT}`);
  });
};

startServer();
