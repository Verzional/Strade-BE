import { DataSource } from "typeorm";
import { Skill } from "../models/skillModel.js";
import { UserSkill } from "../models/userSkillModel.js";
import dotenv from "dotenv";

dotenv.config();

if (!process.env.DB_URI) {
  throw new Error("CRITICAL ERROR: DB_URI environment variable is missing.");
}

export const AppDataSource = new DataSource({
  type: "postgres",
  url: process.env.DB_URI,
  synchronize: true, // Only for PoC/Development
  logging: false,
  entities: [Skill, UserSkill],
});

export const connectDB = async () => {
  try {
    await AppDataSource.initialize();
    console.log("PostgreSQL Connected via TypeORM (SkillService)");
  } catch (error) {
    console.error("Error connecting to PostgreSQL:", error);
    process.exit(1);
  }
};
