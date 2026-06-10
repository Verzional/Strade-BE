// services/user/src/routes/userRoutes.ts
import { Router } from "express";
import {
  getPublicProfile,
  getUserProfile,
  getAllUsers,
} from "../controllers/authController.js";
import { verifyToken } from "../middlewares/authMiddleware.js";

const router: Router = Router();

router.get("/profile", verifyToken, getUserProfile);
router.get("/all", verifyToken, getAllUsers); // This MUST come before /:id
router.get("/:id", getPublicProfile);

export default router;
