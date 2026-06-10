import express from "express";
import cors from "cors";
import skillRoutes from "./routes/skillRoutes.js";

const app = express();

// app.use(cors());
app.use(express.json());

app.use("/api/skills", skillRoutes);

export default app;
