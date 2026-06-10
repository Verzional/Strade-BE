import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn, OneToMany, type Relation } from 'typeorm';
import { Account } from './accountModel.js';
import { Session } from './sessionModel.js';

@Entity('users')
export class User {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column({ nullable: true })
  name?: string;

  @Column({ unique: true, nullable: true })
  email?: string;

  @Column({ nullable: true })
  image?: string;

  @Column({ type: 'timestamp', nullable: true })
  emailVerified?: Date;

  // TypeORM Relations
  @OneToMany(() => Account, (account) => account.user)
  accounts!: Relation<Account>[];

  @OneToMany(() => Session, (session) => session.user)
  sessions!: Relation<Session>[];

  @CreateDateColumn()
  createdAt!: Date;

  @UpdateDateColumn()
  updatedAt!: Date;
}