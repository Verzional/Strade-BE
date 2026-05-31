import "reflect-metadata";
import { AppDataSource } from "./config/db.js";
import { Skill } from "./models/skillModel.js";
import { UserSkill } from "./models/userSkillModel.js";
import dotenv from "dotenv";

dotenv.config();

const seed = async () => {
  await AppDataSource.initialize();
  console.log("Connected to DB for seeding...");

  const skillRepo = AppDataSource.getRepository(Skill);
  const userSkillRepo = AppDataSource.getRepository(UserSkill);

  // Clear existing
  await userSkillRepo.createQueryBuilder().delete().execute();
  await skillRepo.createQueryBuilder().delete().execute();

  // 1. Seed Skills
  const baseSkills = ["TypeScript", "Node.js", "React", "Figma", "Python"].map(
    (name) => {
      const s = new Skill();
      s.skillName = name;
      return s;
    },
  );

  const savedSkills = await skillRepo.save(baseSkills);
  console.log("Seeded Skills!");

  // 2. Seed Mock User Skills (For PoC)
  // Generating a fake valid UUID for a dummy user matching what a gateway might forward

  const dummyUserId1 = "11111111-1111-1111-1111-111111111111";
  const dummyUserId2 = "22222222-2222-2222-2222-222222222222";

  const tsSkill = savedSkills.find((s) => s.skillName === "TypeScript");

  if (tsSkill) {
    const us1 = new UserSkill();
    us1.user_id = dummyUserId1;
    us1.username = "Alice The Teacher";
    us1.email = "alice@example.com";
    us1.image = "https://ui-avatars.com/api/?name=Alice";
    us1.skill_id = tsSkill.id;
    us1.is_teaching = true;

    const us2 = new UserSkill();
    us2.user_id = dummyUserId2;
    us2.username = "Bob The Learner";
    us2.email = "bob@example.com";
    us2.image = "https://ui-avatars.com/api/?name=Bob";
    us2.skill_id = tsSkill.id;
    us2.is_teaching = false;

    await userSkillRepo.save([us1, us2]);
    console.log("Seeded UserSkills with denormalized profile data!");
  }

  process.exit(0);
};

seed().catch((err) => {
  console.error(err);
  process.exit(1);
});
