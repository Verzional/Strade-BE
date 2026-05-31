import { Request, Response } from "express";
import { getAllSkills, getMatchedUsers } from "../services/skillService.js";
import { matchSkillsSchema } from "../validators/skillValidator.js";

export const getSkills = async (req: Request, res: Response) => {
  try {
    const skills = await getAllSkills();
    res.status(200).json(skills);
  } catch (error) {
    console.error(error);
    res.status(500).json({ error: "Internal Server Error" });
  }
};

export const matchSkills = async (
  req: Request,
  res: Response,
): Promise<void> => {
  try {
    const validation = matchSkillsSchema.safeParse(req.query);
    if (!validation.success) {
      res.status(400).json({ error: validation.error.errors });
      return;
    }

    const { skill_id, intent } = validation.data;

    // Direct database fetch, no network requests needed
    const matchedUsers = await getMatchedUsers(skill_id, intent);
    res.status(200).json(matchedUsers);
  } catch (error) {
    console.error(error);
    res.status(500).json({ error: "Internal Server Error" });
  }
};
