import {
  Entity,
  PrimaryGeneratedColumn,
  Column,
  CreateDateColumn,
  UpdateDateColumn,
  ManyToOne,
  JoinColumn,
  Index,
} from "typeorm";
import { Skill } from "./skillModel.js";

@Entity("user_skills")
@Index(["user_id", "skill_id", "is_teaching"], { unique: true })
export class UserSkill {
  @PrimaryGeneratedColumn("uuid")
  id!: string;

  @Column({ type: "uuid" })
  user_id!: string;

  @ManyToOne(() => Skill)
  @JoinColumn({ name: "skill_id" })
  skill!: Skill;

  @Column({ type: "uuid" })
  skill_id!: string;

  // Denormalized User Data
  @Column({ type: "varchar" })
  username!: string;

  @Column({ type: "varchar", nullable: true })
  email!: string;

  @Column({ type: "varchar", nullable: true })
  image!: string;

  @Column({ type: "boolean" })
  is_teaching!: boolean;

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}
