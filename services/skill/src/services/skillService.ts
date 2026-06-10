import { AppDataSource } from "../config/db.js";
import { Skill } from "../models/skillModel.js";
import { UserSkill } from "../models/userSkillModel.js";

export const getAllSkills = async () => {
  const skillRepo = AppDataSource.getRepository(Skill);
  return await skillRepo.find();
};

export const getMatchedUsers = async (
  skill_id: string,
  intent: "learn" | "teach",
) => {
  const userSkillRepo = AppDataSource.getRepository(UserSkill);

  // If user wants to learn, find users who are teaching (true).
  // If user wants to teach, find users who are learning (false).
  const targetTeaching = intent === "learn" ? true : false;

  const matchedUserSkills = await userSkillRepo.find({
    where: { skill_id, is_teaching: targetTeaching },
    relations: ["skill"],
  });

  if (matchedUserSkills.length === 0) return [];

  // Return the denormalized data directly without calling UserService
  return matchedUserSkills.map((us) => ({
    user_id: us.user_id,
    username: us.username,
    email: us.email,
    image: us.image,
    skillName: us.skill.skillName,
    is_teaching: us.is_teaching,
  }));
};
