import { Router } from "express";
import { getSkills, matchSkills } from "../controllers/skillController.js";

const router = Router();

router.get("/", getSkills);
router.get("/match", matchSkills);

export default router;
